# Thread Architecture Diagram

## System Overview

```mermaid
graph TB
    Client[Client] --> WS[WebSocket Connection]
    Client --> HTTP[HTTP API]

    WS --> Room[Room Manager]
    WS --> Thread[Thread Manager]

    HTTP --> Room
    HTTP --> Thread

    Room --> Store[(In-Memory Store)]
    Thread --> Store

    Store --> Users[Users Map]
    Store --> Rooms[Rooms Map]
    Store --> Threads[Threads Map]

    subgraph "Room Structure"
        Room1[Room]
        Room1 --> RoomMsgs[Messages Array]
        Room1 --> RoomMembers[Members Map]
        Room1 --> RoomConns[Connections Map]
    end

    subgraph "Thread Structure"
        Thread1[Thread]
        Thread1 --> ThreadMsgs[Messages Array]
        Thread1 --> ThreadMembers[Members Map]
        Thread1 --> ThreadConns[Connections Map]
        Thread1 --> ThreadRoom[Parent Room ID]
    end
```

## Thread Creation Flow

```mermaid
sequenceDiagram
    participant C as Client
    participant WS as WebSocket
    participant S as Server
    participant R as Room
    participant T as Thread
    participant ST as Store

    Note over C,ST: Create Thread from Message
    C->>WS: create_thread {roomId, name, rootMessageId}
    WS->>S: handleCreateThread()
    S->>ST: getRoom(roomId)
    ST-->>S: Room object
    S->>R: findMessage(rootMessageId)
    R-->>S: Message object
    S->>T: createThread()
    T->>ST: store thread
    S->>R: update message.ThreadID
    S->>WS: broadcast new_thread to room
    WS->>C: new_thread event
    S->>T: auto-join creator
    S->>WS: thread_joined event
    WS->>C: thread_joined event
```

## Thread Message Flow

```mermaid
sequenceDiagram
    participant C as Client
    participant WS as WebSocket
    participant S as Server
    participant T as Thread
    participant ST as Store

    Note over C,ST: Send Message to Thread
    C->>WS: send_thread_message {threadId, content, replyTo}
    WS->>S: handleSendThreadMessage()
    S->>ST: getThread(threadId)
    ST-->>S: Thread object
    S->>T: validate membership
    S->>T: create message with ThreadID
    T->>ST: store message in thread
    S->>WS: broadcast new_thread_message to thread
    WS->>C: new_thread_message event
```

## Thread Join/Leave Flow

```mermaid
sequenceDiagram
    participant C as Client
    participant WS as WebSocket
    participant S as Server
    participant T as Thread
    participant R as Room
    participant ST as Store

    Note over C,ST: Join Thread
    C->>WS: join_thread {threadId}
    WS->>S: joinThread()
    S->>ST: getThread(threadId)
    ST-->>S: Thread object
    S->>R: validate room membership
    S->>T: add user to members
    S->>T: add connection to thread
    S->>WS: thread_joined event
    WS->>C: thread_joined event
    S->>WS: broadcast user_joined_thread
    WS->>C: user_joined_thread event

    Note over C,ST: Leave Thread
    C->>WS: leave_thread {}
    WS->>S: leaveThread()
    S->>T: remove connection
    S->>T: remove member
    S->>WS: broadcast user_left_thread
    WS->>C: user_left_thread event
```

## Data Relationships

```mermaid
erDiagram
    User ||--o{ RoomMember : joins
    User ||--o{ ThreadMember : joins
    User ||--o{ Message : sends
    Room ||--o{ Message : contains
    Room ||--o{ Thread : hosts
    Thread ||--o{ Message : contains
    Message ||--o{ Message : replies_to
    Message ||--o{ Reaction : has
    Message ||--o{ Seen : tracked_by
    User ||--o{ Reaction : adds
    User ||--o{ Seen : marks

    User {
        string AuthKey PK
        string Username
        time CreatedAt
        time LastSeen
    }

    Room {
        string ID PK
        string Name
        string Description
        string CreatedBy
        time CreatedAt
        map Members
        array Messages
        map Connections
    }

    Thread {
        string ID PK
        string RoomID FK
        string Name
        string RootMessageId
        string CreatedBy
        time CreatedAt
        map Members
        array Messages
        map Connections
    }

    Message {
        string ID PK
        string RoomId FK
        string ThreadId FK
        string SenderAuthKey FK
        string Content
        time Timestamp
        string Type
        string ReplyToId FK
        map Reactions
        map Seen
    }
```

## WebSocket Event Flow

```mermaid
graph LR
    subgraph "Client Events"
        CE1[create_thread]
        CE2[join_thread]
        CE3[leave_thread]
        CE4[send_thread_message]
    end

    subgraph "Server Events"
        SE1[new_thread]
        SE2[thread_joined]
        SE3[user_joined_thread]
        SE4[user_left_thread]
        SE5[new_thread_message]
    end

    CE1 --> SE1
    CE1 --> SE2
    CE2 --> SE2
    CE2 --> SE3
    CE3 --> SE4
    CE4 --> SE5

    subgraph "Broadcast Targets"
        BT1[Room Members]
        BT2[Thread Members]
        BT3[Requesting Client]
    end

    SE1 --> BT1
    SE2 --> BT3
    SE3 --> BT2
    SE4 --> BT2
    SE5 --> BT2
```

## HTTP API Endpoints

```mermaid
graph TB
    subgraph "Thread Management"
        GET1[GET /api/rooms/:roomId/threads]
        POST1[POST /api/rooms/:roomId/threads]
        GET2[GET /api/threads/:threadId]
        GET3[GET /api/threads/:threadId/messages]
    end

    subgraph "Thread Operations"
        REACT1[PUT /api/threads/:threadId/messages/:messageId/reactions/:emoji]
        REACT2[DELETE /api/threads/:threadId/messages/:messageId/reactions/:emoji]
        SEEN1[PUT /api/threads/:threadId/messages/:messageId/seen]
    end

    GET1 --> List[List threads in room]
    POST1 --> Create[Create new thread]
    GET2 --> Details[Get thread details]
    GET3 --> Messages[Get thread messages]

    REACT1 --> AddReact[Add reaction]
    REACT2 --> RemoveReact[Remove reaction]
    SEEN1 --> MarkSeen[Mark as seen]
```

## Frontend Component Structure

```mermaid
graph TB
    subgraph "Main Layout"
        Header[Header]
        RoomsPanel[Rooms Panel]
        ChatArea[Chat Area]
    end

    subgraph "Thread Components"
        ThreadList[Thread List]
        ThreadItem[Thread Item]
        ThreadView[Thread View]
        ThreadHeader[Thread Header]
        ThreadMessages[Thread Messages]
        ThreadInput[Thread Input]
    end

    subgraph "Thread Controls"
        CreateThreadBtn[Create Thread Button]
        ThreadIndicator[Thread Indicator]
        JoinThreadBtn[Join Thread Button]
        LeaveThreadBtn[Leave Thread Button]
    end

    RoomsPanel --> ThreadList
    ThreadList --> ThreadItem
    ChatArea --> ThreadView
    ThreadView --> ThreadHeader
    ThreadView --> ThreadMessages
    ThreadView --> ThreadInput

    ThreadHeader --> LeaveThreadBtn
    ThreadMessages --> ThreadIndicator
    ThreadItem --> JoinThreadBtn
    ChatArea --> CreateThreadBtn
```

## State Management Flow

```mermaid
stateDiagram-v2
    [*] --> RoomView
    RoomView --> ThreadList: Click thread
    RoomView --> CreateThread: Click "New Thread"
    ThreadList --> ThreadView: Select thread
    CreateThread --> ThreadView: Thread created
    ThreadView --> RoomView: Click "Back to Room"
    ThreadView --> ThreadView: Send message
    ThreadView --> ThreadView: Add reaction
    ThreadView --> ThreadView: Mark seen

    RoomView: Viewing room messages
    ThreadList: Browsing threads
    CreateThread: Creating new thread
    ThreadView: Active in thread
```

## Error Handling Flow

```mermaid
graph TB
    subgraph "Client Errors"
        CE1[Invalid JSON]
        CE2[Missing Fields]
        CE3[Unauthorized]
    end

    subgraph "Server Errors"
        SE1[Thread Not Found]
        SE2[Room Not Found]
        SE3[Message Not Found]
        SE4[Not Member]
    end

    subgraph "Error Responses"
        ER1[create_thread_error]
        ER2[join_thread_error]
        ER3[send_thread_message_error]
        ER4[thread_error]
    end

    CE1 --> ER1
    CE2 --> ER1
    CE3 --> ER2

    SE1 --> ER2
    SE2 --> ER2
    SE3 --> ER3
    SE4 --> ER2

    ER1 --> Client[Client Error Handler]
    ER2 --> Client
    ER3 --> Client
    ER4 --> Client
```

## Performance Considerations

```mermaid
graph LR
    subgraph "Memory Usage"
        M1[Thread Messages]
        M2[Member Maps]
        M3[Connection Maps]
    end

    subgraph "Optimization Strategies"
        O1[Message History Limits]
        O2[Connection Cleanup]
        O3[Efficient Broadcasting]
    end

    subgraph "Scalability Factors"
        S1[Threads per Room]
        S2[Members per Thread]
        S3[Messages per Thread]
    end

    M1 --> O1
    M2 --> O2
    M3 --> O3

    O1 --> S1
    O2 --> S2
    O3 --> S3
```

This architecture provides a comprehensive threading system that integrates seamlessly with the existing room-based chat while maintaining performance and scalability.
