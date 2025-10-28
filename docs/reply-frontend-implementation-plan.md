# Reply Functionality Frontend Implementation Plan

## Overview

This document outlines the implementation plan for adding reply functionality to the chat.html frontend, following the WhatsApp/Discord-style reply system with preview and visual indicators.

## Architecture

### Data Flow

1. User clicks reply button on a message
2. Reply preview appears in message input area
3. User types message and sends
4. Message includes replyTo ID sent to backend
5. Backend returns message with replyTo preview
6. Frontend displays message with reply preview

### UI Components

1. **Reply Button**: Small button on each message to initiate reply
2. **Reply Preview**: Shows original message being replied to
3. **Reply Indicator**: Visual line/indentation showing reply relationship
4. **Cancel Reply**: X button to cancel reply operation

## Implementation Details

### 1. CSS Styles for Reply UI Components

#### Reply Preview Container

```css
.reply-preview {
  background-color: #f0f2f5;
  border-left: 3px solid #4a6fa5;
  padding: 0.5rem;
  margin-bottom: 0.5rem;
  border-radius: 4px;
  font-size: 0.9rem;
}

.reply-preview-content {
  color: #666;
  font-style: italic;
}

.reply-preview-sender {
  font-weight: bold;
  color: #4a6fa5;
  margin-right: 0.5rem;
}
```

#### Reply Button

```css
.reply-btn {
  position: absolute;
  top: 0.5rem;
  right: 0.5rem;
  background-color: transparent;
  border: 1px solid #ddd;
  border-radius: 4px;
  padding: 0.25rem 0.5rem;
  font-size: 0.8rem;
  cursor: pointer;
  opacity: 0;
  transition: opacity 0.2s;
}

.message-container:hover .reply-btn {
  opacity: 1;
}

.reply-btn:hover {
  background-color: #f0f0f0;
}
```

#### Reply Indicator in Messages

```css
.message-reply-indicator {
  border-left: 2px solid #4a6fa5;
  margin-left: 0.5rem;
  padding-left: 0.5rem;
}

.message-reply-preview {
  background-color: #f8f9fa;
  border-radius: 4px;
  padding: 0.5rem;
  margin-bottom: 0.5rem;
  font-size: 0.85rem;
  color: #666;
}
```

### 2. State Management Variables

```javascript
// Reply state management
let replyingTo = null; // { messageId, sender, content }
let replyMessageId = null; // ID of message being replied to
let isReplying = false; // Flag to track reply state
```

### 3. Message Display Updates

#### Update addMessage Function

```javascript
function addMessage(
  sender,
  content,
  timestamp,
  isOwn = false,
  messageId = null,
  reactions = [],
  seen = { count: 0, users: [] },
  replyTo = null // NEW: reply information
) {
  const messageContainer = document.createElement("div");
  messageContainer.className = `message-container`;

  const messageDiv = document.createElement("div");
  messageDiv.className = `message ${isOwn ? "own" : ""}`;
  messageDiv.setAttribute("data-message-id", messageId);

  const time = new Date(timestamp).toLocaleTimeString();

  // Build message HTML with reply preview
  let messageHTML = `
    <div class="message-info">${sender} • ${time}</div>
  `;

  // Add reply preview if exists
  if (replyTo && replyTo.id) {
    messageHTML += `
      <div class="message-reply-preview">
        <span class="reply-preview-sender">Replying to ${replyTo.sender}</span>
        <div class="reply-preview-content">${escapeHtml(replyTo.content)}</div>
      </div>
    `;
  }

  messageHTML += `
    <div class="message-content">${escapeHtml(content)}</div>
    <div class="message-reactions"></div>
    <div class="message-seen"></div>
    <div class="reaction-picker">
      ${REACTION_EMOJIS.map(
        (emoji) =>
          `<span class="reaction-option" data-emoji="${emoji}">${emoji}</span>`
      ).join("")}
    </div>
  `;

  messageDiv.innerHTML = messageHTML;

  // Add reply button
  const replyBtn = document.createElement("button");
  replyBtn.className = "reply-btn";
  replyBtn.textContent = "Reply";
  replyBtn.onclick = () => startReply(messageId, sender, content);
  messageDiv.appendChild(replyBtn);

  messageContainer.appendChild(messageDiv);
  messagesContainer.appendChild(messageContainer);

  // ... rest of existing logic for reactions, seen, etc.
}
```

### 4. Reply Functionality Implementation

#### Start Reply Function

```javascript
function startReply(messageId, sender, content) {
  replyingTo = { messageId, sender, content };
  replyMessageId = messageId;
  isReplying = true;

  // Update message input placeholder
  messageInput.placeholder = `Replying to ${sender}...`;

  // Show reply preview in input area
  showReplyPreview();

  // Focus on input
  messageInput.focus();
}
```

#### Show Reply Preview Function

```javascript
function showReplyPreview() {
  if (!isReplying || !replyingTo) return;

  // Remove existing reply preview if any
  const existingPreview = document.querySelector(".reply-input-preview");
  if (existingPreview) {
    existingPreview.remove();
  }

  // Create reply preview element
  const replyPreview = document.createElement("div");
  replyPreview.className = "reply-input-preview";
  replyPreview.innerHTML = `
    <div class="reply-preview">
      <span class="reply-preview-sender">Replying to ${replyingTo.sender}</span>
      <div class="reply-preview-content">${escapeHtml(replyingTo.content)}</div>
      <button class="reply-cancel-btn" onclick="cancelReply()">✕</button>
    </div>
  `;

  // Insert before message input container
  const inputContainer = document.querySelector(".message-input-container");
  inputContainer.parentNode.insertBefore(replyPreview, inputContainer);
}
```

#### Cancel Reply Function

```javascript
function cancelReply() {
  replyingTo = null;
  replyMessageId = null;
  isReplying = false;

  // Reset placeholder
  messageInput.placeholder = "Type your message...";

  // Remove reply preview
  const replyPreview = document.querySelector(".reply-input-preview");
  if (replyPreview) {
    replyPreview.remove();
  }

  // Focus on input
  messageInput.focus();
}
```

### 5. Update Message Sending

#### Update sendMessage Function

```javascript
function sendMessage() {
  const content = messageInput.value.trim();
  if (content && currentRoomId) {
    const messageData = {
      roomId: currentRoomId,
      content: content,
    };

    // Add replyTo if replying
    if (isReplying && replyMessageId) {
      messageData.replyTo = replyMessageId;
    }

    sendWebSocketMessage("send_message", messageData);
    messageInput.value = "";

    // Clear reply state after sending
    if (isReplying) {
      cancelReply();
    }
  } else if (!currentRoomId) {
    addSystemMessage("Please select a room first");
  }
}
```

### 6. Update Message Handling

#### Update handleNewMessage Function

```javascript
function handleNewMessage(data) {
  const messageData = data; // Data is already parsed by WebSocket
  addMessage(
    messageData.sender,
    messageData.content,
    messageData.timestamp,
    messageData.sender === username,
    messageData.id,
    messageData.reactions || [],
    messageData.seen || { count: 0, users: [] },
    messageData.replyTo || null // NEW: pass reply information
  );
}
```

#### Update loadMessages Function

```javascript
// In the messages.forEach loop:
addMessage(
  message.sender,
  message.content,
  message.timestamp,
  message.sender === username,
  message.id,
  message.reactions || [],
  message.seen || { count: 0, users: [] },
  message.replyTo || null // NEW: pass reply information
);
```

### 7. CSS for Reply Input Preview

```css
.reply-input-preview {
  margin-bottom: 0.5rem;
  padding: 0 1rem;
}

.reply-input-preview .reply-preview {
  position: relative;
  background-color: #f0f2f5;
  border-left: 3px solid #4a6fa5;
  padding: 0.5rem;
  border-radius: 4px;
  font-size: 0.9rem;
}

.reply-cancel-btn {
  position: absolute;
  top: 0.5rem;
  right: 0.5rem;
  background: none;
  border: none;
  font-size: 1rem;
  cursor: pointer;
  color: #666;
}

.reply-cancel-btn:hover {
  color: #333;
}
```

## Implementation Steps

1. **Add CSS Styles**: Add all reply-related CSS styles to the existing style section
2. **Add State Variables**: Add reply state management variables at the top of the script
3. **Update addMessage**: Modify to handle and display reply information
4. **Update Message Handlers**: Update handleNewMessage and loadMessages to pass reply data
5. **Add Reply Functions**: Implement startReply, showReplyPreview, cancelReply functions
6. **Update sendMessage**: Modify to include replyTo information when sending
7. **Add Reply Button**: Add reply button to each message in addMessage function
8. **Test Integration**: Test the complete reply functionality with the backend

## Testing Plan

1. **Basic Reply Flow**: Test clicking reply button, seeing preview, sending reply
2. **Reply Display**: Verify replies show with proper preview and indicators
3. **Cancel Reply**: Test canceling a reply operation
4. **Multiple Replies**: Test multiple replies in conversation
5. **Reply to Own Message**: Test replying to own messages
6. **Reply History**: Verify replies show correctly when loading message history

## Backend Integration Notes

The backend already supports:

- `ReplyToID` field in Message struct
- `replyTo` field in WebSocket message payload
- `replyPreview` struct for reply information
- Reply validation in `handleSendMessage`

The frontend needs to:

- Send `replyTo` in message payload
- Handle `replyTo` in received messages
- Display reply preview information
