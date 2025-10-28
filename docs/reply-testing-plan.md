# Reply Message Mechanism Testing Plan

## Overview

This document outlines testing procedures for the reply message mechanism to ensure it works correctly with both HTTP API and WebSocket connections.

## Test Scenarios

### 1. Basic Reply Functionality

#### 1.1 WebSocket Reply Test

**Objective**: Verify users can reply to messages via WebSocket

**Steps**:

1. Connect two clients (Alice and Bob) to the same room via WebSocket
2. Alice sends a message: "Hello everyone"
3. Bob replies to Alice's message with: "Hi Alice!"
4. Verify both clients receive the reply with correct reply preview

**Expected Results**:

- Bob's message includes `replyTo` field with Alice's message preview
- Reply preview contains: id, sender, trimmed content, timestamp, type
- Both Alice and Bob receive the reply message

#### 1.2 HTTP API Reply Test

**Objective**: Verify replies appear correctly in HTTP message history

**Steps**:

1. Send a message with a reply via WebSocket
2. Call `GET /api/rooms/:roomId/messages`
3. Verify the response includes reply information

**Expected Results**:

- Message in HTTP response includes `replyTo` field
- Reply preview matches the WebSocket broadcast format

### 2. Validation Tests

#### 2.1 Invalid Reply Target Test

**Objective**: Verify proper error handling for non-existent reply targets

**Steps**:

1. Connect a client to a room
2. Attempt to reply to a non-existent message ID
3. Verify error response

**Expected Results**:

- Client receives error: "replyTo message not found in this room"
- No message is created or broadcast

#### 2.2 Empty Reply Target Test

**Objective**: Verify normal message sending when replyTo is empty

**Steps**:

1. Send a message with empty `replyTo` field
2. Verify message is processed normally

**Expected Results**:

- Message is created without reply information
- No `replyTo` field in the broadcast

### 3. Edge Cases

#### 3.1 Reply to System Message Test

**Objective**: Verify replies work with system messages

**Steps**:

1. Trigger a system message (user joins room)
2. Reply to the system message
3. Verify reply is processed correctly

**Expected Results**:

- Reply is created successfully
- Reply preview shows system message details
- System message type is preserved in preview

#### 3.2 Reply to Reply Test

**Objective**: Verify one-level threading (no nested replies)

**Steps**:

1. Send original message
2. Reply to original message
3. Reply to the reply message
4. Verify all messages are handled correctly

**Expected Results**:

- All messages are created successfully
- Each reply shows preview of its direct parent
- No special handling for "reply to reply" (flat structure)

### 4. Performance Tests

#### 4.1 Large Reply Chain Test

**Objective**: Verify performance with multiple replies

**Steps**:

1. Create a message with many replies
2. Verify message loading performance
3. Check reply preview generation efficiency

**Expected Results**:

- Message loading remains responsive
- Reply previews are generated quickly
- No memory leaks or excessive resource usage

### 5. Concurrency Tests

#### 5.1 Simultaneous Replies Test

**Objective**: Verify thread safety with concurrent replies

**Steps**:

1. Have multiple users reply to the same message simultaneously
2. Verify all replies are processed correctly
3. Check for race conditions

**Expected Results**:

- All replies are created successfully
- No data corruption or inconsistent state
- Proper locking behavior

## Test Data

### Sample WebSocket Messages

#### Send Reply Message:

```json
{
  "event": "send_message",
  "data": {
    "roomId": "room-abc123",
    "content": "I agree with this point!",
    "replyTo": "msg-def456"
  }
}
```

#### Expected New Message Broadcast:

```json
{
  "event": "new_message",
  "data": {
    "id": "msg-ghi789",
    "roomId": "room-abc123",
    "sender": "Alice",
    "content": "I agree with this point!",
    "timestamp": "2025-10-28T01:30:00Z",
    "type": "text",
    "reactions": [],
    "seen": {
      "count": 1,
      "users": ["Alice"]
    },
    "replyTo": {
      "id": "msg-def456",
      "sender": "Bob",
      "content": "This is the original message that might be quite long and needs to be...",
      "timestamp": "2025-10-28T01:25:00Z",
      "type": "text"
    }
  }
}
```

### Sample HTTP Response

#### GET /api/rooms/room-abc123/messages:

```json
[
  {
    "id": "msg-def456",
    "roomId": "room-abc123",
    "sender": "bob-auth-key",
    "content": "This is the original message that might be quite long and needs to be truncated for the preview.",
    "timestamp": "2025-10-28T01:25:00Z",
    "type": "text",
    "reactions": [],
    "seen": {
      "count": 2,
      "users": ["Bob", "Alice"]
    },
    "replyTo": null
  },
  {
    "id": "msg-ghi789",
    "roomId": "room-abc123",
    "sender": "alice-auth-key",
    "content": "I agree with this point!",
    "timestamp": "2025-10-28T01:30:00Z",
    "type": "text",
    "reactions": [
      {
        "emoji": "👍",
        "count": 1,
        "reactors": ["Bob"]
      }
    ],
    "seen": {
      "count": 1,
      "users": ["Alice"]
    },
    "replyTo": {
      "id": "msg-def456",
      "sender": "Bob",
      "content": "This is the original message that might be quite long and needs to be...",
      "timestamp": "2025-10-28T01:25:00Z",
      "type": "text"
    }
  }
]
```

## Test Automation

### Unit Tests

1. Test `makeReplyPreview()` function with various message types
2. Test `replyPreviewByID()` with valid and invalid IDs
3. Test `trimRunes()` utility function
4. Test `handleSendMessage()` with and without replyTo

### Integration Tests

1. End-to-end WebSocket reply flow
2. HTTP API message retrieval with replies
3. Cross-client reply synchronization
4. Error handling scenarios

### Load Tests

1. High volume of replies in a short time
2. Multiple concurrent users replying
3. Memory usage with large reply chains

## Success Criteria

1. ✅ All basic reply functionality works as expected
2. ✅ Error handling is robust and user-friendly
3. ✅ Performance remains acceptable under load
4. ✅ Thread safety is maintained
5. ✅ Backward compatibility is preserved
6. ✅ Both WebSocket and HTTP APIs support replies
