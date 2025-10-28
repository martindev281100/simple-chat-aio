# Seen Mechanism Implementation Plan

## Overview

This plan outlines the implementation of a simplified "seen" mechanism for the chat application. The feature will track which users have read specific messages and provide this information through both HTTP API and WebSocket events.

## Requirements

- Track who has seen each message (with timestamps)
- HTTP endpoint: `PUT /api/rooms/:roomId/messages/:messageId/seen`
- WS event: `mark_seen` (client → server) and broadcast `seen_updated` (server → clients)
- `GET /api/rooms/:roomId/messages` includes a `seen` summary per message
- Sender is marked as having seen their own message on send

## Data Model Changes

### 1. Add SeenState to Message struct

```go
type SeenState map[string]time.Time // authKey -> seenAt

type Message struct {
    ID            string        `json:"id"`
    RoomID        string        `json:"roomId"`
    SenderAuthKey string        `json:"sender"`
    Content       string        `json:"content"`
    Timestamp     time.Time     `json:"timestamp"`
    Type          string        `json:"type"` // "text", "system"
    Reactions     ReactionState `json:"-"`    // emoji -> {authKey: true}
    Seen          SeenState     `json:"-"`    // authKey -> seenAt
}
```

### 2. Create seenSummary type for API responses

```go
type seenSummary struct {
    Count int      `json:"count"`
    Users []string `json:"users,omitempty"` // usernames
}
```

## Core Implementation

### 3. Implement summarizeSeen method

Converts SeenState to seenSummary for API responses:

```go
func (a *app) summarizeSeen(m *Message) seenSummary {
    if m.Seen == nil || len(m.Seen) == 0 {
        return seenSummary{Count: 0, Users: nil}
    }
    users := make([]string, 0, len(m.Seen))
    for ak := range m.Seen {
        users = append(users, a.store.username(ak))
    }
    return seenSummary{Count: len(users), Users: users}
}
```

### 4. Add findMessageIndex method

Locates messages by index for efficient operations:

```go
func (a *app) findMessageIndex(room *Room, messageID string) int {
    for i := range room.Messages {
        if room.Messages[i].ID == messageID {
            return i
        }
    }
    return -1
}
```

### 5. Implement setSeen method

Marks a message as seen by a user:

```go
func (a *app) setSeen(room *Room, messageID, authKey string) (bool, []string, int, error) {
    now := time.Now()
    room.mu.Lock()
    defer room.mu.Unlock()

    idx := a.findMessageIndex(room, messageID)
    if idx == -1 {
        return false, nil, 0, errors.New("message not found")
    }

    msg := &room.Messages[idx]
    if msg.Seen == nil {
        msg.Seen = make(SeenState)
    }

    prev, ok := msg.Seen[authKey]
    msg.Seen[authKey] = now
    changed := !ok || prev.Before(now)

    users := make([]string, 0, len(msg.Seen))
    for ak := range msg.Seen {
        users = append(users, a.store.username(ak))
    }

    return changed, users, len(users), nil
}
```

### 6. Add broadcastSeen method

Notifies clients about seen updates:

```go
func (a *app) broadcastSeen(room *Room, messageID string, count int, users []string) {
    payload, _ := json.Marshal(map[string]any{
        "roomId":    room.ID,
        "messageId": messageID,
        "count":     count,
        "users":     users, // usernames
    })
    a.broadcast(room, wsEnvelope{
        Event: "seen_updated",
        Data:  payload,
    }, nil)
}
```

## HTTP API Implementation

### 7. Add markSeenHTTP handler

```go
func (a *app) markSeenHTTP(c *gin.Context) {
    u := getUser(c)
    roomID := c.Param("roomId")
    messageID := c.Param("messageId")

    room, ok := a.store.getRoom(roomID)
    if !ok {
        c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
        return
    }

    changed, users, count, err := a.setSeen(room, messageID, u.AuthKey)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    if changed {
        a.broadcastSeen(room, messageID, count, users)
    }
    c.JSON(http.StatusOK, gin.H{
        "roomId":    room.ID,
        "messageId": messageID,
        "count":     count,
        "users":     users,
    })
}
```

### 8. Add HTTP route

Add to the API group in main():

```go
// Seen
api.PUT("/rooms/:roomId/messages/:messageId/seen", app.markSeenHTTP)
```

## WebSocket Implementation

### 9. Add handleMarkSeen method

```go
func (c *wsConn) handleMarkSeen(roomID, messageID string) {
    if c.roomID == "" || c.roomID != roomID {
        c.joinRoom(roomID)
        if c.roomID != roomID {
            return
        }
    }
    room, ok := c.app.store.getRoom(roomID)
    if !ok {
        c.sendError("mark_seen_error", "room not found")
        return
    }
    changed, users, count, err := c.app.setSeen(room, messageID, c.authKey)
    if err != nil {
        c.sendError("mark_seen_error", err.Error())
        return
    }
    if changed {
        c.app.broadcastSeen(room, messageID, count, users)
    }
}
```

### 10. Add "mark_seen" event to WebSocket readLoop

```go
case "mark_seen":
    var p struct {
        RoomID    string `json:"roomId"`
        MessageID string `json:"messageId"`
    }
    if err := json.Unmarshal(env.Data, &p); err != nil {
        c.sendError("mark_seen_error", "bad json")
        continue
    }
    c.handleMarkSeen(p.RoomID, p.MessageID)
```

## Integration with Existing Features

### 11. Update getMessages to include seen information

Add seen field to messageResp:

```go
type messageResp struct {
    ID        string         `json:"id"`
    RoomID    string         `json:"roomId"`
    Sender    string         `json:"sender"`
    Content   string         `json:"content"`
    Timestamp time.Time      `json:"timestamp"`
    Type      string         `json:"type"`
    Reactions []reactionResp `json:"reactions,omitempty"`
    Seen      seenSummary    `json:"seen"`
}
```

Update the response construction:

```go
out = append(out, messageResp{
    ID:        m.ID,
    RoomID:    m.RoomID,
    Sender:    m.SenderAuthKey,
    Content:   m.Content,
    Timestamp: m.Timestamp,
    Type:      m.Type,
    Reactions: a.summarizeReactions(m),
    Seen:      a.summarizeSeen(m),
})
```

### 12. Update handleSendMessage to mark sender as seen

```go
msg := Message{
    ID:            "msg-" + shortID(),
    RoomID:        room.ID,
    SenderAuthKey: c.authKey,
    Content:       content,
    Timestamp:     time.Now(),
    Type:          "text",
    Seen:          SeenState{c.authKey: time.Now()}, // Mark sender as seen
}
```

Update the broadcast data:

```go
data := map[string]any{
    "id":        msg.ID,
    "roomId":    msg.RoomID,
    "sender":    c.username,
    "content":   msg.Content,
    "timestamp": msg.Timestamp,
    "reactions": []reactionResp{},
    "seen":      seenSummary{Count: 1, Users: []string{c.username}},
}
```

### 13. Update addSystemMessage

System messages should have empty seen state initially:

```go
msg := Message{
    ID:            "sys-" + shortID(),
    RoomID:        room.ID,
    SenderAuthKey: "system",
    Content:       content,
    Timestamp:     time.Now(),
    Type:          "system",
    Seen:          SeenState{}, // Empty seen state for system messages
}
```

## Testing

### 14. Test Implementation

- Test HTTP endpoint for marking messages as seen
- Test WebSocket event for marking messages as seen
- Verify seen information is included in message lists
- Verify broadcast events are sent correctly
- Test edge cases (non-existent messages, unauthorized access)

## API Usage Examples

### HTTP

```bash
# Mark message as seen
PUT /api/rooms/room-123/messages/msg-456/seen
# Response: {"roomId":"room-123","messageId":"msg-456","count":2,"users":["Alice","Bob"]}
```

### WebSocket

```json
// Client sends
{"event":"mark_seen","data":{"roomId":"room-123","messageId":"msg-456"}}

// Server broadcasts
{"event":"seen_updated","data":{"roomId":"room-123","messageId":"msg-456","count":2,"users":["Alice","Bob"]}}
```

## Implementation Order

1. Data model changes (1-2)
2. Core helper methods (3-6)
3. HTTP API implementation (7-8)
4. WebSocket implementation (9-10)
5. Integration with existing features (11-13)
6. Testing (14)

This plan provides a comprehensive approach to implementing the seen mechanism while maintaining consistency with the existing codebase architecture.
