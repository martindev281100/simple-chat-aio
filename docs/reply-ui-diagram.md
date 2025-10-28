# Reply Functionality UI Diagram

## Message Display with Reply

```
┌─────────────────────────────────────────────────────────────┐
│ [Avatar] User A • 10:30 AM                    [Reply] │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ This is the original message being replied to        │ │
│ └─────────────────────────────────────────────────────────┘ │
│ [Reactions: 👍(2) ❤️(1)] [Seen: 👁️ 3]             │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│ [Avatar] User B • 10:32 AM                    [Reply] │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ │ Replying to User A                                │ │
│ │ │ This is the original message being replied to...    │ │
│ └─────────────────────────────────────────────────────────┘ │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ This is the reply message from User B                │ │
│ └─────────────────────────────────────────────────────────┘ │
│ [Reactions: 👍(1)] [Seen: 👁️ 2]                     │
└─────────────────────────────────────────────────────────────┘
```

## Reply Interaction Flow

```
┌─────────────┐    Click Reply    ┌──────────────────┐    Type Message    ┌─────────────┐
│ Original    │ ────────────────► │ Reply Preview    │ ────────────────► │ Send Reply  │
│ Message     │                  │ in Input Area   │                  │ Message     │
│ with Reply  │                  │                 │                  │ with replyTo │
│ Button      │                  │                 │                  │ ID          │
└─────────────┘                  └──────────────────┘                  └─────────────┘
        │                                   │                                   │
        │                                   │                                   │
        ▼                                   ▼                                   ▼
┌─────────────┐                  ┌──────────────────┐                  ┌─────────────┐
│ Show Reply  │                  │ User can cancel  │                  │ Backend     │
│ Preview     │                  │ reply with X     │                  │ validates   │
│ Component   │                  │ button           │                  │ replyTo ID  │
└─────────────┘                  └──────────────────┘                  └─────────────┘
```

## UI Components Layout

### Message Container with Reply

```
┌─────────────────────────────────────────────────────────────┐
│ message-container                                         │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ message                                             │ │
│ │ ┌─────────────────────────────────────────────────┐   │ │
│ │ │ message-info                                  │   │ │
│ │ │ [User] • [Time]                      [Reply] │   │ │
│ │ └─────────────────────────────────────────────────┘   │ │
│ │                                                     │ │
│ │ ┌─────────────────────────────────────────────────┐   │ │
│ │ │ message-reply-preview (if replyTo exists)     │   │ │
│ │ │ Replying to [Original User]                  │   │ │
│ │ │ [Original message content...]                  │   │ │
│ │ └─────────────────────────────────────────────────┘   │ │
│ │                                                     │ │
│ │ ┌─────────────────────────────────────────────────┐   │ │
│ │ │ message-content                              │   │ │
│ │ │ [Current message content]                    │   │ │
│ │ └─────────────────────────────────────────────────┘   │ │
│ │                                                     │ │
│ │ [Reactions] [Seen] [Reaction Picker]                  │ │
│ └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### Reply Input Area

```
┌─────────────────────────────────────────────────────────────┐
│ reply-input-preview (when replying)                      │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ reply-preview                                      │ │
│ │ ┌─────────────────────────────────────────────────┐   │ │
│ │ │ Replying to [User]                          │   │ │
│ │ │ [Original message content...]                │   │ │
│ │ └─────────────────────────────────────────────────┘   │ │
│ │                                          [X Cancel] │ │
│ └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────────┐
│ message-input-container                                   │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ message-input                                     │ │ │
│ │ [Type your reply...]                              │ │ │
│ └─────────────────────────────────────────────────────────┘ │
│ [Send Button]                                           │
└─────────────────────────────────────────────────────────────┘
```

## State Management Flow

```
┌─────────────────┐    Click Reply    ┌─────────────────┐
│ Normal State    │ ────────────────► │ Reply State     │
│                 │                  │                 │
│ - replyingTo    │                  │ - replyingTo    │
│   = null        │                  │   = {id,       │
│ - replyMessageId│                  │       sender,   │
│   = null        │                  │       content}  │
│ - isReplying    │                  │ - replyMessageId│
│   = false       │                  │   = messageId   │
│                 │                  │ - isReplying    │
└─────────────────┘                  │   = true       │
        ▲                           │                 │
        │ Cancel Reply                └─────────────────┘
        │                                   │
        │                                   ▼ Send Message
        │                          ┌─────────────────┐
        │                          │ Normal State    │
        └───────────────────────── │                 │
                                   │ - replyingTo    │
                                   │   = null        │
                                   │ - replyMessageId│
                                   │   = null        │
                                   │ - isReplying    │
                                   │   = false       │
                                   └─────────────────┘
```

## CSS Class Hierarchy

```
message-container
├── message
│   ├── message-info
│   ├── message-reply-preview (if replyTo exists)
│   │   ├── reply-preview-sender
│   │   └── reply-preview-content
│   ├── message-content
│   ├── message-reactions
│   ├── message-seen
│   ├── reaction-picker
│   └── reply-btn
└── (other elements)

reply-input-preview (when replying)
└── reply-preview
    ├── reply-preview-sender
    ├── reply-preview-content
    └── reply-cancel-btn
```

## Event Handling

```
Message Events:
- click reply-btn → startReply(messageId, sender, content)
- hover message-container → show reply-btn
- leave message-container → hide reply-btn

Input Events:
- click reply-cancel-btn → cancelReply()
- submit message → sendMessage() with replyTo
- keypress Enter → sendMessage() with replyTo

WebSocket Events:
- receive new_message with replyTo → addMessage() with replyTo
- send_message with replyTo → backend processes reply
```
