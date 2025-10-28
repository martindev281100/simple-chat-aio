# Reply Message Mechanism Implementation Plan

## Overview

This document outlines the implementation of a reply/quote mechanism for the chat application, allowing users to reply to specific messages with one-level threading.

## Data Model Changes

### 1. Message Struct Update

Add `ReplyToID` field to the `Message` struct:

```go
type Message struct {
    ID            string        `json:"id"`
    RoomID        string        `json:"roomId"`
    SenderAuthKey string        `json:"sender"`
    Content       string        `json:"content"`
    Timestamp     time.Time     `json:"timestamp"`
    Type          string        `json:"type"` // "text", "system"
    Reactions     ReactionState `json:"-"`    // emoji -> {authKey: true}
    Seen          SeenState     `json:"-"`    // authKey -> seenAt

    // NEW: id of the message this one replies to (if any)
    ReplyToID string `json:"replyTo,omitempty"`
}
```

### 2. Reply Preview DTO

Add a new struct for reply preview:

```go
type replyPreview struct {
    ID        string    `json:"id"`
    Sender    string    `json:"sender"`   // username (not authKey)
    Content   string    `json:"content"`  // trimmed
    Timestamp time.Time `json:"timestamp"`
    Type      string    `json:"type"`
}
```

## Helper Functions

### 1. Reply Preview Builders

```go
func (a *app) makeReplyPreview(room *Room, m *Message) *replyPreview {
    if m == nil {
        return nil
    }
    return &replyPreview{
        ID:        m.ID,
        Sender:    a.store.username(m.SenderAuthKey),
        Content:   trimRunes(m.Content, 120),
        Timestamp: m.Timestamp,
        Type:      m.Type,
    }
}

func (a *app) replyPreviewByID(room *Room, id string) *replyPreview {
    if strings.TrimSpace(id) == "" {
        return nil
    }
    room.mu.RLock()
    defer room.mu.RUnlock()
    m := a.findMessage(room, id)
    return a.makeReplyPreview(room, m)
}
```

### 2. Utility Function

```go
func trimRunes(s string, n int) string {
    r := []rune(s)
    if len(r) <= n {
        return s
    }
    return string(r[:n])
}
```

## HTTP API Changes

### 1. Update messageResp Struct

Add ReplyTo field to the response struct:

```go
type messageResp struct {
    ID        string         `json:"id"`
    RoomID    string         `json:"roomId"`
    Sender    string         `json:"sender"` // authKey (kept compat with old API)
    Content   string         `json:"content"`
    Timestamp time.Time      `json:"timestamp"`
    Type      string         `json:"type"`
    Reactions []reactionResp `json:"reactions,omitempty"`
    Seen      seenSummary    `json:"seen"`

    // NEW
    ReplyTo *replyPreview `json:"replyTo,omitempty"`
}
```

### 2. Update getMessages Handler

Modify the loop that builds the response to include reply preview:

```go
for i := start; i < end; i++ {
    m := &room.Messages[i]
    var rp *replyPreview
    if m.ReplyToID != "" {
        rp = a.replyPreviewByID(room, m.ReplyToID)
    }
    out = append(out, messageResp{
        ID:        m.ID,
        RoomID:    m.RoomID,
        Sender:    m.SenderAuthKey,
        Content:   m.Content,
        Timestamp: m.Timestamp,
        Type:      m.Type,
        Reactions: a.summarizeReactions(m),
        Seen:      a.summarizeSeen(m),
        ReplyTo:   rp, // NEW
    })
}
```

## WebSocket Changes

### 1. Update send_message Payload

Add ReplyTo field to the payload:

```go
case "send_message":
    var payload struct {
        RoomID  string `json:"roomId"`
        Content string `json:"content"`
        ReplyTo string `json:"replyTo"` // NEW (optional)
    }
    if err := json.Unmarshal(env.Data, &payload); err != nil {
        c.sendError("send_message_error", "bad json")
        continue
    }
    c.handleSendMessage(strings.TrimSpace(payload.RoomID), strings.TrimSpace(payload.Content), strings.TrimSpace(payload.ReplyTo))
```

### 2. Update handleSendMessage Signature

Change the function signature to accept replyTo:

```go
func (c *wsConn) handleSendMessage(roomID, content, replyTo string)
```

### 3. Update handleSendMessage Implementation

Replace the existing implementation with:

```go
func (c *wsConn) handleSendMessage(roomID, content, replyTo string) {
    if content == "" {
        c.sendError("send_message_error", "content required")
        return
    }
    // must be in the room (auto-join like before)
    if c.roomID == "" || c.roomID != roomID {
        c.joinRoom(roomID)
        if c.roomID != roomID {
            return
        }
    }
    room, ok := c.app.store.getRoom(roomID)
    if !ok {
        c.sendError("send_message_error", "room not found")
        return
    }

    // Validate reply target (if provided)
    var rp *replyPreview
    if replyTo != "" {
        room.mu.RLock()
        target := c.app.findMessage(room, replyTo)
        if target == nil {
            room.mu.RUnlock()
            c.sendError("send_message_error", "replyTo message not found in this room")
            return
        }
        rp = c.app.makeReplyPreview(room, target)
        room.mu.RUnlock()
    }

    msg := Message{
        ID:            "msg-" + shortID(),
        RoomID:        room.ID,
        SenderAuthKey: c.authKey,
        Content:       content,
        Timestamp:     time.Now(),
        Type:          "text",
        Seen:          SeenState{c.authKey: time.Now()}, // sender has seen
        ReplyToID:     replyTo,                           // store linkage
    }

    room.mu.Lock()
    room.Messages = append(room.Messages, msg)
    if len(room.Messages) > room.maxHist {
        room.Messages = room.Messages[len(room.Messages)-room.maxHist:]
    }
    room.mu.Unlock()

    data := map[string]any{
        "id":        msg.ID,
        "roomId":    msg.RoomID,
        "sender":    c.username, // username in WS broadcasts (unchanged)
        "content":   msg.Content,
        "timestamp": msg.Timestamp,
        "reactions": []reactionResp{},
        "seen":      seenSummary{Count: 1, Users: []string{c.username}},
        "replyTo":   rp, // NEW – already a preview struct or nil
    }
    payload, _ := json.Marshal(data)
    c.app.broadcast(room, wsEnvelope{
        Event: "new_message",
        Data:  payload,
    }, nil)
}
```

## Client Usage Examples

### WebSocket send_message (no reply):

```json
{
  "event": "send_message",
  "data": { "roomId": "room-abc", "content": "hello" }
}
```

### WebSocket send_message (with reply):

```json
{
  "event": "send_message",
  "data": { "roomId": "room-abc", "content": "agree!", "replyTo": "msg-xyz" }
}
```

### WebSocket new_message broadcast (with reply):

```json
{
  "event": "new_message",
  "data": {
    "id": "msg-123",
    "roomId": "room-abc",
    "sender": "Alice",
    "content": "agree!",
    "timestamp": "2025-10-28T01:23:45Z",
    "reactions": [],
    "seen": { "count": 1, "users": ["Alice"] },
    "replyTo": {
      "id": "msg-xyz",
      "sender": "Bob",
      "content": "Original message trimmed…",
      "timestamp": "2025-10-28T01:22:10Z",
      "type": "text"
    }
  }
}
```

## Implementation Order

1. Update data models (Message struct, add replyPreview)
2. Add helper functions (makeReplyPreview, replyPreviewByID, trimRunes)
3. Update HTTP API (messageResp, getMessages handler)
4. Update WebSocket (payload, handler signature, implementation)
5. Test the implementation

## Notes

- All changes are backward-compatible
- Reply mechanism works with both HTTP and WebSocket APIs
- Thread-safe with existing room locks
- Allows replies to all messages (including system messages)
- Reply content is trimmed to 120 characters for preview
