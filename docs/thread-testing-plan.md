# Thread Implementation Testing Plan

## Overview

This testing plan ensures the Discord-style thread implementation works correctly with existing room functionality while maintaining backward compatibility and performance.

## Testing Environment Setup

### Prerequisites

- Go 1.25+ installed
- WebSocket-compatible browser
- Test users with different auth keys
- Network monitoring tools (browser dev tools, Wireshark)

### Test Data

- Multiple test rooms
- Test messages with various content types
- Test users with different permission levels
- Pre-existing room messages for thread creation

## Unit Tests

### 1. Thread Model Tests

#### Test: Thread Creation

```go
func TestCreateThread(t *testing.T) {
    store := NewStore()
    room := store.createRoom("Test Room", "Description", "user1")

    // Test thread creation without root message
    thread, err := store.createThread(room.ID, "Test Thread", "user1", "")
    assert.NoError(t, err)
    assert.Equal(t, room.ID, thread.RoomID)
    assert.Equal(t, "Test Thread", thread.Name)
    assert.Empty(t, thread.RootMessageID)

    // Test thread creation with root message
    thread2, err := store.createThread(room.ID, "Reply Thread", "user1", "msg-123")
    assert.NoError(t, err)
    assert.Equal(t, "msg-123", thread2.RootMessageID)
}
```

#### Test: Thread Retrieval

```go
func TestGetThread(t *testing.T) {
    store := NewStore()
    room := store.createRoom("Test Room", "Description", "user1")
    thread, _ := store.createThread(room.ID, "Test Thread", "user1", "")

    retrieved, exists := store.getThread(thread.ID)
    assert.True(t, exists)
    assert.Equal(t, thread.ID, retrieved.ID)

    _, exists = store.getThread("non-existent")
    assert.False(t, exists)
}
```

#### Test: Thread Listing

```go
func TestListThreads(t *testing.T) {
    store := NewStore()
    room1 := store.createRoom("Room 1", "Description", "user1")
    room2 := store.createRoom("Room 2", "Description", "user1")

    thread1, _ := store.createThread(room1.ID, "Thread 1", "user1", "")
    thread2, _ := store.createThread(room1.ID, "Thread 2", "user1", "")
    thread3, _ := store.createThread(room2.ID, "Thread 3", "user1", "")

    room1Threads := store.listThreads(room1.ID)
    assert.Len(t, room1Threads, 2)

    room2Threads := store.listThreads(room2.ID)
    assert.Len(t, room2Threads, 1)
}
```

### 2. WebSocket Thread Tests

#### Test: Thread Creation via WebSocket

```go
func TestCreateThreadWebSocket(t *testing.T) {
    app := newApp()
    store := app.store
    room := store.createRoom("Test Room", "Description", "user1")

    // Simulate WebSocket connection
    conn := &wsConn{
        authKey:  "user1",
        username: "user1",
        roomID:   room.ID,
        app:      app,
    }

    // Test thread creation
    payload := map[string]interface{}{
        "roomId": room.ID,
        "name":   "Test Thread",
    }

    conn.handleCreateThread(room.ID, "Test Thread", "")

    // Verify thread was created
    threads := store.listThreads(room.ID)
    assert.Len(t, threads, 1)
    assert.Equal(t, "Test Thread", threads[0].Name)
}
```

#### Test: Thread Join/Leave via WebSocket

```go
func TestThreadJoinLeaveWebSocket(t *testing.T) {
    app := newApp()
    store := app.store
    room := store.createRoom("Test Room", "Description", "user1")
    thread, _ := store.createThread(room.ID, "Test Thread", "user1", "")

    conn := &wsConn{
        authKey:  "user2",
        username: "user2",
        roomID:   room.ID,
        app:      app,
    }

    // Test joining thread
    conn.joinThread(thread.ID)
    assert.Equal(t, thread.ID, conn.threadID)

    thread.mu.RLock()
    assert.Contains(t, thread.Members, "user2")
    thread.mu.RUnlock()

    // Test leaving thread
    conn.leaveThread()
    assert.Empty(t, conn.threadID)

    thread.mu.RLock()
    assert.NotContains(t, thread.Members, "user2")
    thread.mu.RUnlock()
}
```

### 3. Thread Message Tests

#### Test: Thread Message Creation

```go
func TestThreadMessageCreation(t *testing.T) {
    app := newApp()
    store := app.store
    room := store.createRoom("Test Room", "Description", "user1")
    thread, _ := store.createThread(room.ID, "Test Thread", "user1", "")

    conn := &wsConn{
        authKey:  "user1",
        username: "user1",
        roomID:   room.ID,
        threadID: thread.ID,
        app:      app,
    }

    // Test sending message to thread
    conn.handleSendThreadMessage(thread.ID, "Hello Thread", "")

    thread.mu.RLock()
    assert.Len(t, thread.Messages, 1)
    assert.Equal(t, "Hello Thread", thread.Messages[0].Content)
    assert.Equal(t, thread.ID, thread.Messages[0].ThreadID)
    thread.mu.RUnlock()
}
```

## Integration Tests

### 1. HTTP API Tests

#### Test: List Threads Endpoint

```bash
# Test listing threads in a room
curl -X GET "http://localhost:8080/api/rooms/room-123/threads" \
  -H "X-Auth-Key: user1-auth" \
  -H "X-Username: user1"

# Expected response:
[
  {
    "id": "thr-abc123",
    "roomId": "room-123",
    "name": "Test Thread",
    "rootMessageId": "msg-456",
    "createdBy": "user1",
    "createdAt": "2023-01-01T12:00:00Z",
    "memberCount": 2
  }
]
```

#### Test: Create Thread Endpoint

```bash
# Test creating a thread
curl -X POST "http://localhost:8080/api/rooms/room-123/threads" \
  -H "Content-Type: application/json" \
  -H "X-Auth-Key: user1-auth" \
  -H "X-Username: user1" \
  -d '{
    "name": "New Thread",
    "rootMessageId": "msg-789"
  }'

# Expected response:
{
  "id": "thr-def456",
  "roomId": "room-123",
  "name": "New Thread",
  "rootMessageId": "msg-789",
  "createdBy": "user1",
  "createdAt": "2023-01-01T12:00:00Z"
}
```

#### Test: Get Thread Messages Endpoint

```bash
# Test getting thread messages
curl -X GET "http://localhost:8080/api/threads/thr-abc123/messages?limit=10&offset=0" \
  -H "X-Auth-Key: user1-auth" \
  -H "X-Username: user1"

# Expected response:
[
  {
    "id": "msg-111",
    "roomId": "room-123",
    "threadId": "thr-abc123",
    "sender": "user1",
    "content": "First thread message",
    "timestamp": "2023-01-01T12:00:00Z",
    "type": "text",
    "reactions": [],
    "seen": {"count": 1, "users": ["user1"]},
    "replyTo": null
  }
]
```

### 2. WebSocket Integration Tests

#### Test: Complete Thread Workflow

```javascript
// Test complete thread creation and messaging workflow
const ws = new WebSocket(
  "ws://localhost:8080/ws?authKey=test-auth&username=testuser"
);

// 1. Join room
ws.send(
  JSON.stringify({
    event: "join_room",
    data: { roomId: "room-123" },
  })
);

// 2. Create thread from message
ws.send(
  JSON.stringify({
    event: "create_thread",
    data: {
      roomId: "room-123",
      name: "Discussion Thread",
      rootMessageId: "msg-456",
    },
  })
);

// 3. Join thread
ws.send(
  JSON.stringify({
    event: "join_thread",
    data: { threadId: "thr-abc123" },
  })
);

// 4. Send message to thread
ws.send(
  JSON.stringify({
    event: "send_thread_message",
    data: {
      threadId: "thr-abc123",
      content: "This is a thread message",
      replyTo: "",
    },
  })
);

// 5. Leave thread
ws.send(
  JSON.stringify({
    event: "leave_thread",
    data: {},
  })
);
```

## Manual Testing Scenarios

### 1. Basic Thread Operations

#### Scenario 1: Create Thread from Message

1. **Setup**: User in room with existing messages
2. **Action**: Click "Create Thread" on a message
3. **Expected**:
   - Thread creation modal appears
   - Thread created with message as root
   - Original message shows thread indicator
   - User auto-joins new thread
   - Room members notified of new thread

#### Scenario 2: Create Standalone Thread

1. **Setup**: User in room
2. **Action**: Click "New Thread" button
3. **Expected**:
   - Thread creation modal appears
   - Thread created without root message
   - User auto-joins new thread
   - Thread appears in room's thread list

#### Scenario 3: Join/Leave Thread

1. **Setup**: User in room with existing thread
2. **Action**: Click thread in list, then click "Leave Thread"
3. **Expected**:
   - User joins thread successfully
   - Thread messages load
   - User appears in thread member list
   - Leaving returns user to room view
   - Other members notified of join/leave

### 2. Thread Message Operations

#### Scenario 4: Send Thread Message

1. **Setup**: User in thread
2. **Action**: Type and send message
3. **Expected**:
   - Message appears in thread view
   - Message has thread ID
   - Other thread members receive message
   - Room members don't receive message
   - Message supports replies and reactions

#### Scenario 5: Reply to Thread Message

1. **Setup**: User in thread with existing messages
2. **Action**: Click reply on a message, send reply
3. **Expected**:
   - Reply preview shown in input
   - Reply sent with replyTo ID
   - Reply appears with original message reference
   - Reply chain maintained within thread

### 3. Thread Reactions and Seen

#### Scenario 6: Thread Message Reactions

1. **Setup**: User in thread with messages
2. **Action**: Add reaction to thread message
3. **Expected**:
   - Reaction appears on message
   - Other thread members see reaction
   - Reaction count updates correctly
   - Can remove reaction

#### Scenario 7: Thread Message Seen Status

1. **Setup**: User in thread with unread messages
2. **Action**: Scroll through messages
3. **Expected**:
   - Messages marked as seen when visible
   - Seen count updates for senders
   - Seen indicators show who has read
   - Real-time seen updates

### 4. Error Handling

#### Scenario 8: Invalid Thread Operations

1. **Setup**: User not in room
2. **Action**: Try to join thread directly
3. **Expected**:
   - Error message displayed
   - User not added to thread
   - Appropriate error response

#### Scenario 9: Thread Permission Validation

1. **Setup**: User not in room
2. **Action**: Try to create thread in room
3. **Expected**:
   - Creation fails
   - Error message about room access
   - No thread created

## Performance Tests

### 1. Load Testing

#### Test: Multiple Concurrent Threads

```bash
# Create 100 threads concurrently
for i in {1..100}; do
  curl -X POST "http://localhost:8080/api/rooms/room-123/threads" \
    -H "Content-Type: application/json" \
    -H "X-Auth-Key: user$i-auth" \
    -H "X-Username: user$i" \
    -d "{\"name\": \"Thread $i\"}" &
done
wait

# Verify all threads created
curl -X GET "http://localhost:8080/api/rooms/room-123/threads" \
  -H "X-Auth-Key: user1-auth" | jq '. | length'
```

#### Test: Thread Message Volume

```bash
# Send 1000 messages to a thread
for i in {1..1000}; do
  curl -X POST "http://localhost:8080/api/threads/thr-abc123/messages" \
    -H "Content-Type: application/json" \
    -H "X-Auth-Key: user1-auth" \
    -d "{\"content\": \"Message $i\"}" &
done
wait
```

### 2. Memory Usage Tests

#### Test: Thread Memory Footprint

```go
func TestThreadMemoryUsage(t *testing.T) {
    app := newApp()
    store := app.store
    room := store.createRoom("Test Room", "Description", "user1")

    // Create many threads
    for i := 0; i < 1000; i++ {
        store.createThread(room.ID, fmt.Sprintf("Thread %d", i), "user1", "")
    }

    // Check memory usage
    var m runtime.MemStats
    runtime.ReadMemStats(&m)

    // Verify reasonable memory usage
    assert.Less(t, m.Alloc, uint64(100*1024*1024)) // Less than 100MB
}
```

## Regression Tests

### 1. Backward Compatibility

#### Test: Existing Room Functionality

1. **Setup**: Existing room with messages
2. **Action**: Use all existing room features
3. **Expected**:
   - Room messaging works unchanged
   - Reactions work unchanged
   - Seen status works unchanged
   - No performance degradation

#### Test: Mixed Room/Thread Usage

1. **Setup**: Room with both regular messages and threads
2. **Action**: Use room and thread features simultaneously
3. **Expected**:
   - No interference between room and thread messages
   - Proper isolation of thread messages
   - Room broadcasts don't affect threads

### 2. Edge Cases

#### Test: Empty Thread Names

```bash
curl -X POST "http://localhost:8080/api/rooms/room-123/threads" \
  -H "Content-Type: application/json" \
  -H "X-Auth-Key: user1-auth" \
  -d '{"name": ""}'

# Expected: Auto-generated thread name
```

#### Test: Very Long Thread Names

```bash
curl -X POST "http://localhost:8080/api/rooms/room-123/threads" \
  -H "Content-Type: application/json" \
  -H "X-Auth-Key: user1-auth" \
  -d '{"name": "'$(printf 'a%.0s' {1..1000})'"}'

# Expected: Name accepted or truncated appropriately
```

## Test Automation

### 1. Automated Test Suite

```bash
# Run all thread tests
go test -v ./... -run TestThread

# Run with race detection
go test -race -v ./... -run TestThread

# Run with coverage
go test -cover -v ./... -run TestThread
```

### 2. Continuous Integration

```yaml
# .github/workflows/thread-tests.yml
name: Thread Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: 1.25
      - run: go test -v -race -cover ./...
      - run: go test -bench=. ./...
```

## Test Results Documentation

### 1. Test Report Template

```
Thread Implementation Test Report
================================

Environment:
- Go Version: 1.25.1
- Browser: Chrome 120.0
- Date: 2023-01-01

Test Results:
- Unit Tests: 45/45 passed
- Integration Tests: 12/12 passed
- Manual Tests: 15/15 passed
- Performance Tests: 8/8 passed

Issues Found:
- None critical
- 2 minor UI issues documented

Performance Metrics:
- Thread creation: <50ms
- Thread join: <30ms
- Thread message: <20ms
- Memory usage: <2MB per 100 threads

Recommendations:
- Ready for production deployment
- Monitor thread creation rate
- Consider thread cleanup policy
```

This comprehensive testing plan ensures the thread implementation is robust, performant, and maintains backward compatibility with existing functionality.
