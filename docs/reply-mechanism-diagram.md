# Reply Message Mechanism Flow Diagram

## Message Reply Flow

```mermaid
sequenceDiagram
    participant Client
    participant WebSocket
    participant Server
    participant Room
    participant MessageStore

    Client->>WebSocket: send_message with replyTo
    WebSocket->>Server: handleSendMessage(roomID, content, replyTo)

    alt replyTo is provided
        Server->>Room: RLock to find target message
        Room->>MessageStore: findMessage(replyTo)
        MessageStore-->>Room: Message or nil
        alt message not found
            Server-->>Client: Error: replyTo message not found
        else message found
            Server->>Server: makeReplyPreview(target)
            Server->>Room: RUnlock
        end
    end

    Server->>Room: Lock to add new message
    Server->>MessageStore: Create Message with ReplyToID
    MessageStore-->>Room: Message stored
    Room->>Room: Unlock

    Server->>Server: Create broadcast data with reply preview
    Server->>Room: broadcast(new_message)
    Room->>Client: new_message event with replyTo data
```

## Data Structure Changes

```mermaid
classDiagram
    class Message {
        +string ID
        +string RoomID
        +string SenderAuthKey
        +string Content
        +time.Time Timestamp
        +string Type
        +ReactionState Reactions
        +SeenState Seen
        +string ReplyToID
    }

    class replyPreview {
        +string ID
        +string Sender
        +string Content
        +time.Time Timestamp
        +string Type
    }

    class messageResp {
        +string ID
        +string RoomID
        +string Sender
        +string Content
        +time.Time Timestamp
        +string Type
        +reactionResp[] Reactions
        +seenSummary Seen
        +replyPreview ReplyTo
    }

    Message "1" --> "0..1" replyPreview : generates
    messageResp "1" --> "0..1" replyPreview : includes
```

## WebSocket Message Flow

```mermaid
flowchart TD
    A[Client sends send_message] --> B{Has replyTo?}
    B -->|No| C[Process as regular message]
    B -->|Yes| D[Validate replyTo exists]
    D --> E{Message found?}
    E -->|No| F[Send error response]
    E -->|Yes| G[Create reply preview]
    G --> H[Store message with ReplyToID]
    C --> H
    H --> I[Broadcast new_message with replyTo]
    I --> J[All clients receive message with reply preview]
```

## HTTP API Response Structure

```mermaid
json
{
  "messages": [
    {
      "id": "msg-123",
      "roomId": "room-abc",
      "sender": "user-auth-key",
      "content": "This is a reply",
      "timestamp": "2025-10-28T01:23:45Z",
      "type": "text",
      "reactions": [],
      "seen": {
        "count": 2,
        "users": ["Alice", "Bob"]
      },
      "replyTo": {
        "id": "msg-456",
        "sender": "Bob",
        "content": "Original message content that might be very long...",
        "timestamp": "2025-10-28T01:22:10Z",
        "type": "text"
      }
    }
  ]
}
```

## Implementation Components

1. **Data Model Updates**

   - Add `ReplyToID` to `Message` struct
   - Add `replyPreview` struct for API responses

2. **Helper Functions**

   - `makeReplyPreview()` - Creates preview from message
   - `replyPreviewByID()` - Safely retrieves preview by ID
   - `trimRunes()` - Utility for content trimming

3. **HTTP API Changes**

   - Update `messageResp` to include `ReplyTo` field
   - Modify `getMessages` handler to populate reply previews

4. **WebSocket Changes**

   - Update `send_message` payload to accept `replyTo`
   - Modify `handleSendMessage` to validate and store replies
   - Include reply preview in `new_message` broadcasts

5. **Validation Logic**
   - Ensure reply target exists in the same room
   - Handle cases where reply target is not found
