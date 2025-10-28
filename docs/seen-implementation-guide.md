# Seen Mechanism Implementation Guide

This guide provides step-by-step implementation details for adding the seen mechanism to the chat application.

## Step 1: Add Data Models

### 1.1 Add SeenState to Message struct

**Location**: Line 31-40 in main.go

**Current code**:

```go
type Message struct {
	ID            string        `json:"id"`
	RoomID        string        `json:"roomId"`
	SenderAuthKey string        `json:"sender"`
	Content       string        `json:"content"`
	Timestamp     time.Time     `json:"timestamp"`
	Type          string        `json:"type"` // "text", "system"
	Reactions     ReactionState `json:"-"`    // emoji -> {authKey: true}
}
```

**Updated code**:

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
}
```

### 1.2 Add SeenState and seenSummary types

**Location**: After line 61 (after ReactionState definition)

**Add this code**:

```go
type SeenState map[string]time.Time // authKey -> seenAt

type seenSummary struct {
	Count int      `json:"count"`
	Users []string `json:"users,omitempty"` // usernames
}
```

## Step 2: Add Helper Methods

### 2.1 Add summarizeSeen method

**Location**: After summarizeReactions method (around line 237)

**Add this code**:

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

### 2.2 Add findMessageIndex method

**Location**: After findMessage method (around line 247)

**Add this code**:

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

### 2.3 Add setSeen method

**Location**: After setReaction method (around line 298)

**Add this code**:

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

### 2.4 Add broadcastSeen method

**Location**: After broadcastReaction method (around line 313)

**Add this code**:

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

## Step 3: Add HTTP Handler

### 3.1 Add markSeenHTTP handler

**Location**: After removeReactionHTTP handler (around line 573)

**Add this code**:

```go
func (a *app) markSeenHTTP(c *gin.Context) {
	u := getUser(c)
	roomID := c.Param("roomId")
	messageID := c.Param("messageID")

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

### 3.2 Add HTTP route

**Location**: In main() function, after reaction routes (around line 999)

**Add this code**:

```go
// Seen
api.PUT("/rooms/:roomId/messages/:messageId/seen", app.markSeenHTTP)
```

## Step 4: Update Existing Methods

### 4.1 Update getMessages to include seen information

**Location**: Lines 467-515

**Current messageResp struct**:

```go
type messageResp struct {
	ID        string         `json:"id"`
	RoomID    string         `json:"roomId"`
	Sender    string         `json:"sender"` // authKey (kept compat with old API)
	Content   string         `json:"content"`
	Timestamp time.Time      `json:"timestamp"`
	Type      string         `json:"type"`
	Reactions []reactionResp `json:"reactions,omitempty"`
}
```

**Updated messageResp struct**:

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
}
```

**Update the response construction** (around line 504):

```go
out = append(out, messageResp{
	ID:        m.ID,
	RoomID:    m.RoomID,
	Sender:    m.SenderAuthKey, // unchanged
	Content:   m.Content,
	Timestamp: m.Timestamp,
	Type:      m.Type,
	Reactions: a.summarizeReactions(m),
	Seen:      a.summarizeSeen(m),
})
```

### 4.2 Update handleSendMessage

**Location**: Lines 743-788

**Update message creation** (around line 760):

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

**Update broadcast data** (around line 775):

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

### 4.3 Update addSystemMessage

**Location**: Lines 897-912

**Update message creation** (around line 900):

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

## Step 5: Add WebSocket Support

### 5.1 Add handleMarkSeen method

**Location**: After handleToggleReaction method (around line 849)

**Add this code**:

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

### 5.2 Add "mark_seen" event to WebSocket readLoop

**Location**: In readLoop switch statement (around line 677)

**Add this case**:

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

## Step 6: Testing

### 6.1 Test HTTP Endpoint

```bash
# Test marking a message as seen
curl -X PUT "http://localhost:8080/api/rooms/room-123/messages/msg-456/seen" \
  -H "X-Auth-Key: user123" \
  -H "X-Username: TestUser"
```

### 6.2 Test WebSocket Event

```javascript
// Connect to WebSocket
const ws = new WebSocket(
  "ws://localhost:8080/ws?authKey=user123&username=TestUser"
);

// Send mark_seen event
ws.send(
  JSON.stringify({
    event: "mark_seen",
    data: {
      roomId: "room-123",
      messageId: "msg-456",
    },
  })
);

// Listen for seen_updated event
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  if (data.event === "seen_updated") {
    console.log("Seen updated:", data.data);
  }
};
```

### 6.3 Verify getMessages includes seen information

```bash
# Get messages with seen information
curl "http://localhost:8080/api/rooms/room-123/messages" \
  -H "X-Auth-Key: user123" \
  -H "X-Username: TestUser"
```

## Implementation Checklist

- [ ] Add SeenState field to Message struct
- [ ] Create seenSummary type for API responses
- [ ] Implement summarizeSeen method
- [ ] Add findMessageIndex method
- [ ] Implement setSeen method
- [ ] Add broadcastSeen method
- [ ] Add markSeenHTTP handler
- [ ] Add HTTP route for seen endpoint
- [ ] Update getMessages to include seen information
- [ ] Update handleSendMessage to mark sender as seen
- [ ] Update addSystemMessage to handle seen state
- [ ] Add handleMarkSeen WebSocket method
- [ ] Add "mark_seen" event to WebSocket readLoop
- [ ] Test HTTP endpoint
- [ ] Test WebSocket events
- [ ] Verify integration with existing features

## Common Issues and Solutions

### Issue: Duplicate seen entries

**Solution**: The setSeen method checks if the user has already seen the message and only updates if the timestamp is newer.

### Issue: Race conditions

**Solution**: All message operations use room mutex locks to ensure thread safety.

### Issue: Memory usage

**Solution**: Old messages are automatically pruned based on the room's maxHist setting.

### Issue: Performance with many users

**Solution**: SeenState uses a map for O(1) lookups, and only summary information is broadcasted.

## Next Steps

After implementing the basic seen mechanism, consider these enhancements:

1. **Batch seen operations**: Allow marking multiple messages as seen in one request
2. **Seen receipts privacy**: Add options to control who can see that you've read messages
3. **Seen analytics**: Add endpoints to get seen statistics
4. **Database persistence**: Store seen state in a database for persistence across restarts
