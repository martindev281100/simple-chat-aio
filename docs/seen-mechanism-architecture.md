# Seen Mechanism Architecture

## System Architecture Overview

```mermaid
graph TB
    Client[Chat Client] --> WS[WebSocket Connection]
    Client --> HTTP[HTTP API]

    WS --> WSHandler[WebSocket Handler]
    HTTP --> API[HTTP Handlers]

    WSHandler --> SeenLogic[Seen Logic]
    API --> SeenLogic

    SeenLogic --> Store[In-Memory Store]
    SeenLogic --> Broadcast[Broadcast System]

    Broadcast --> WS
    Broadcast --> OtherClients[Other Connected Clients]

    Store --> Messages[Message Storage]
    Messages --> SeenState[Seen State per Message]
```

## Data Flow for Marking Message as Seen

```mermaid
sequenceDiagram
    participant C as Client
    participant WS as WebSocket Handler
    participant SL as Seen Logic
    participant S as Store
    participant B as Broadcast
    participant OC as Other Clients

    C->>WS: {event: "mark_seen", data: {roomId, messageId}}
    WS->>SL: handleMarkSeen(roomId, messageId)
    SL->>S: setSeen(room, messageId, authKey)
    S-->>SL: (changed, users, count)
    SL->>B: broadcastSeen(room, messageId, count, users)
    B->>OC: {event: "seen_updated", data: {roomId, messageId, count, users}}
    SL-->>WS: Success
    WS-->>C: Acknowledgement
```

## Data Model Relationships

```mermaid
erDiagram
    User {
        string AuthKey
        string Username
        time CreatedAt
        time LastSeen
    }

    Message {
        string ID
        string RoomID
        string SenderAuthKey
        string Content
        time Timestamp
        string Type
        ReactionState Reactions
        SeenState Seen
    }

    Room {
        string ID
        string Name
        string Description
        string CreatedBy
        time CreatedAt
        map Members
        [] Messages
        map conns
    }

    User ||--o{ Message : sends
    Room ||--o{ Message : contains
    Room ||--o{ User : has_members
```

## Component Interaction

### 1. Message Creation Flow

```mermaid
graph LR
    A[Client sends message] --> B[handleSendMessage]
    B --> C[Create Message with SeenState]
    C --> D[Add to Room Messages]
    D --> E[Broadcast to all clients]
    E --> F[Include seen info in broadcast]
```

### 2. Mark as Seen Flow

```mermaid
graph LR
    A[Client marks message seen] --> B[handleMarkSeen/markSeenHTTP]
    B --> C[setSeen method]
    C --> D[Update SeenState]
    D --> E[Check if changed]
    E --> F{Changed?}
    F -->|Yes| G[broadcastSeen]
    F -->|No| H[Return current state]
    G --> I[Notify all clients]
```

## API Endpoints

### HTTP API

- `PUT /api/rooms/:roomId/messages/:messageId/seen`
  - Marks a specific message as seen by the authenticated user
  - Returns updated seen information

### WebSocket Events

- `mark_seen` (Client → Server)

  ```json
  {
    "event": "mark_seen",
    "data": {
      "roomId": "room-123",
      "messageId": "msg-456"
    }
  }
  ```

- `seen_updated` (Server → Clients)
  ```json
  {
    "event": "seen_updated",
    "data": {
      "roomId": "room-123",
      "messageId": "msg-456",
      "count": 3,
      "users": ["Alice", "Bob", "Charlie"]
    }
  }
  ```

## State Management

### SeenState Structure

```go
type SeenState map[string]time.Time // authKey -> seenAt
```

### seenSummary Structure

```go
type seenSummary struct {
    Count int      `json:"count"`
    Users []string `json:"users,omitempty"` // usernames
}
```

## Implementation Considerations

### Thread Safety

- All message operations require room mutex lock
- SeenState updates are atomic within locked context
- Broadcast operations use read locks

### Performance

- SeenState is only included in API responses when explicitly requested
- Broadcast events only include summary information (count and usernames)
- Message lookup by index is O(n) but acceptable for typical chat room sizes

### Memory Usage

- Each message maintains a map of who has seen it
- System messages start with empty seen state
- Old messages are pruned based on room's maxHist setting

## Edge Cases Handled

1. **Non-existent message**: Returns appropriate error
2. **Unauthorized access**: Handled by existing auth middleware
3. **Duplicate seen requests**: Idempotent operation
4. **System messages**: Can be marked as seen like regular messages
5. **User leaves room**: Seen state is preserved for historical messages

## Testing Strategy

### Unit Tests

- Test setSeen method with various scenarios
- Test summarizeSeen conversion
- Test findMessageIndex edge cases

### Integration Tests

- Test HTTP endpoint with authentication
- Test WebSocket event handling
- Test broadcast functionality
- Test concurrent access scenarios

### End-to-End Tests

- Verify seen state updates across multiple clients
- Test message creation with initial seen state
- Verify seen information in message lists
