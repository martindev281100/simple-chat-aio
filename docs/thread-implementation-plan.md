# Discord-Style Threads Implementation Plan

## Overview

This plan outlines the implementation of Discord-style threads for the Go chat server while maintaining backward compatibility with existing room-based messaging. Threads will be contextual conversations that can be created from specific messages or as standalone discussions within rooms.

## Architecture

### Data Model Changes

#### 1. Thread Struct

```go
type Thread struct {
    ID            string           `json:"id"`
    RoomID        string           `json:"roomId"`
    Name          string           `json:"name"`
    RootMessageID string           `json:"rootMessageId,omitempty"` // the message this thread was started from (optional)
    CreatedBy     string           `json:"createdBy"`
    CreatedAt     time.Time        `json:"createdAt"`
    Members       map[string]bool  `json:"-"`
    Messages      []Message        `json:"-"`
    conns         map[*wsConn]bool `json:"-"`
    mu            sync.RWMutex     `json:"-"`
    maxHist       int              `json:"-"`
}
```

#### 2. Message Extension

```go
type Message struct {
    // ... existing fields ...
    ThreadID string `json:"threadId,omitempty"` // NEW: link message to a thread
}
```

#### 3. Store Extension

```go
type Store struct {
    users   map[string]*User
    rooms   map[string]*Room
    threads map[string]*Thread  // NEW: thread storage
    mu      sync.RWMutex
}
```

#### 4. WebSocket Connection Extension

```go
type wsConn struct {
    conn     *websocket.Conn
    writeMu  sync.Mutex
    authKey  string
    username string
    roomID   string
    threadID string // NEW: current thread ID
    app      *app
}
```

## Implementation Phases

### Phase 1: Backend Data Models and Storage

1. **Add Thread data model** (lines ~57-78)

   - Insert Thread struct after Room model
   - Add proper JSON tags and mutex protection

2. **Extend Message struct** (lines ~31-43)

   - Add ThreadID field to link messages to threads
   - Ensure backward compatibility with existing messages

3. **Update Store struct** (lines ~83-87)

   - Add threads map to Store
   - Update NewStore() function to initialize threads map

4. **Implement thread management methods** (lines ~176+)
   - createThread() - Create new thread with optional root message
   - getThread() - Retrieve thread by ID
   - listThreads() - List all threads in a room
   - Thread validation and error handling

### Phase 2: WebSocket Thread Events

1. **Add thread events to readLoop()** (lines ~767-835)

   - create_thread - Create new thread (from message or standalone)
   - join_thread - Join a specific thread
   - leave_thread - Leave current thread
   - send_thread_message - Send message to thread

2. **Implement thread helper methods on wsConn**

   - handleCreateThread() - Process thread creation requests
   - joinThread() - Handle thread joining with auto-room join
   - leaveThread() - Handle thread leaving with cleanup
   - handleSendThreadMessage() - Process thread messages with reply support

3. **Add thread broadcast methods to app**
   - broadcastThread() - Send messages to thread members
   - threadOnlineUsernames() - Get online users in thread
   - findThreadMessage() - Locate message within thread
   - makeReplyPreviewForThread() - Create reply previews for thread messages

### Phase 3: HTTP API Endpoints

1. **Thread management endpoints** (lines ~1189+)

   ```
   GET    /api/rooms/:roomId/threads           - List threads in room
   POST   /api/rooms/:roomId/threads           - Create new thread
   GET    /api/threads/:threadId               - Get thread details
   GET    /api/threads/:threadId/messages      - Get thread messages
   ```

2. **Thread message endpoints** (optional for reactions/seen)
   ```
   PUT    /api/threads/:threadId/messages/:messageId/reactions/:emoji
   DELETE /api/threads/:threadId/messages/:messageId/reactions/:emoji
   PUT    /api/threads/:threadId/messages/:messageId/seen
   ```

### Phase 4: Frontend Integration

1. **Thread UI Components**

   - Thread list panel in room view
   - Thread creation button and modal
   - Thread message view with room context
   - Thread indicators on messages with threads

2. **Thread Event Handling**

   - WebSocket event handlers for thread operations
   - Thread state management
   - Thread message rendering with reactions/seen

3. **User Experience**
   - Seamless switching between room and thread views
   - Thread creation from message context menu
   - Thread notification indicators

## Key Features

### Thread Creation

- **From Message**: Right-click message → "Create Thread" with message as root
- **Standalone**: "New Thread" button in room header
- **Auto-join**: Creator automatically joins created thread

### Thread Management

- **Joining**: Users can join threads they have access to
- **Leaving**: Clean departure with member list updates
- **Membership**: Track online/offline thread members

### Message Features in Threads

- **Replies**: Full reply chain support within threads
- **Reactions**: Same emoji reaction system as room messages
- **Seen Status**: Track which thread members have seen messages
- **History**: Configurable message history limit per thread

### Broadcasting Model

- **Room Broadcasts**: Thread creation/deletion announcements
- **Thread Broadcasts**: Real-time messages to thread members only
- **Presence Updates**: Online user lists for threads

## Implementation Details

### Thread ID Generation

```go
func shortID() string {
    return "thr-" + strconv.FormatInt(time.Now().UnixNano(), 36) + "-" + strconv.FormatInt(int64(rand.Intn(1e6)), 36)
}
```

### Thread Creation Flow

1. Client sends `create_thread` event with roomId, name, optional rootMessageId
2. Server validates room access and root message (if provided)
3. Server creates Thread struct and adds to store
4. Server updates root message with ThreadID (if created from message)
5. Server broadcasts `new_thread` to room members
6. Server auto-joins creator to thread and sends `thread_joined` event

### Thread Message Flow

1. Client sends `send_thread_message` with threadId, content, optional replyTo
2. Server validates thread membership
3. Server creates Message with ThreadID set
4. Server adds message to thread's message slice
5. Server broadcasts `new_thread_message` to thread members

### Thread Join/Leave Flow

1. Client sends `join_thread` or `leave_thread` event
2. Server validates thread existence and room membership
3. Server updates thread member list and connection tracking
4. Server broadcasts membership changes to thread members
5. Server sends confirmation event to requesting client

## Error Handling

### Common Error Cases

- Thread not found
- User not in parent room
- Root message not found (when creating from message)
- Thread message not found (when replying)
- Invalid thread ID format

### Error Response Format

```go
c.sendError("thread_error", "descriptive error message")
```

## Testing Strategy

### Unit Tests

- Thread creation with and without root message
- Thread member management
- Thread message operations
- Thread broadcasting

### Integration Tests

- WebSocket thread event handling
- HTTP thread API endpoints
- Thread creation from room messages
- Thread message reactions and seen status

### Manual Testing

- Create thread from message
- Create standalone thread
- Join/leave thread operations
- Send messages in threads
- Reply to thread messages
- Add reactions to thread messages
- Verify seen status in threads

## Migration Considerations

### Backward Compatibility

- Existing room messages continue to work unchanged
- ThreadID field is optional in Message struct
- Room-based messaging remains fully functional

### Data Migration

- No migration needed for existing data
- New threads are created incrementally
- Existing messages can be linked to threads retroactively if needed

## Performance Considerations

### Memory Usage

- Thread messages stored in memory like room messages
- Configurable history limits per thread
- Connection tracking for online presence

### Scalability

- Thread operations use same locking patterns as rooms
- Broadcasting limited to thread members
- Thread lookups use map for O(1) access

## Security Considerations

### Access Control

- Users must be in parent room to join threads
- Thread creation respects room membership
- Thread messages inherit room permissions

### Validation

- Thread ID format validation
- Root message existence verification
- Room membership verification for thread operations
