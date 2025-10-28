# Thread Frontend Implementation Plan

## Overview

This document outlines the complete implementation plan for adding thread support to the chat.html frontend, matching the thread functionality implemented in main.go backend.

## Current State Analysis

- ✅ Basic room-based chat functionality
- ✅ Reply system (replying to messages)
- ✅ WebSocket connection handling for rooms
- ✅ Message reactions and seen status
- ❌ No thread-related UI or functionality

## Backend Thread Features (Missing in Frontend)

1. **Thread Model**: Discord-style threads within rooms
2. **Thread Creation**: Can create threads from messages or standalone
3. **Thread Management**: Join/leave threads
4. **Thread Messages**: Separate message handling for threads
5. **Thread List**: View all threads in a room
6. **Thread WebSocket Events**: `new_thread`, `thread_joined`, `user_joined_thread`, `new_thread_message`

## Implementation Plan

### 1. UI Components and Styling

#### 1.1 Thread Panel

Add a threads panel next to the rooms panel:

```css
.threads-panel {
  width: 250px;
  background-color: white;
  border-right: 1px solid #ddd;
  display: flex;
  flex-direction: column;
}

.threads-header {
  padding: 1rem;
  border-bottom: 1px solid #eee;
  font-weight: bold;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.threads-list {
  flex: 1;
  overflow-y: auto;
}

.thread-item {
  padding: 0.75rem 1rem;
  cursor: pointer;
  border-bottom: 1px solid #f0f0f0;
  transition: background-color 0.2s;
  font-size: 0.9rem;
}

.thread-item:hover {
  background-color: #f0f0f0;
}

.thread-item.active {
  background-color: #e3f2fd;
  border-left: 3px solid #4a6fa5;
}

.thread-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.thread-name {
  font-weight: 500;
  color: #333;
}

.thread-meta {
  font-size: 0.8rem;
  color: #666;
}

.create-thread-btn {
  margin: 1rem;
  padding: 0.5rem;
  background-color: #4a6fa5;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  transition: background-color 0.2s;
}
```

#### 1.2 Thread Creation Modal

```css
.create-thread-modal {
  display: none;
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background-color: rgba(0, 0, 0, 0.5);
  z-index: 1000;
}

.thread-modal-content {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  background-color: white;
  padding: 2rem;
  border-radius: 8px;
  width: 90%;
  max-width: 400px;
}
```

#### 1.3 Thread Indicators

```css
.thread-indicator {
  display: inline-flex;
  align-items: center;
  padding: 0.2rem 0.5rem;
  background-color: #e3f2fd;
  border-radius: 12px;
  font-size: 0.75rem;
  color: #4a6fa5;
  cursor: pointer;
  transition: background-color 0.2s;
  margin-left: 0.5rem;
}

.thread-indicator:hover {
  background-color: #bbdefb;
}

.thread-count {
  margin-left: 0.25rem;
  font-weight: 500;
}

.message-with-thread {
  position: relative;
}

.thread-start-badge {
  position: absolute;
  top: -0.5rem;
  right: 0.5rem;
  background-color: #4a6fa5;
  color: white;
  font-size: 0.7rem;
  padding: 0.2rem 0.4rem;
  border-radius: 8px;
  cursor: pointer;
}
```

#### 1.4 Thread View Layout

```css
.thread-view {
  display: none;
  flex: 1;
  flex-direction: column;
  background-color: white;
}

.thread-view.active {
  display: flex;
}

.thread-header {
  padding: 1rem;
  border-bottom: 1px solid #eee;
  background-color: #fafafa;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.thread-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.thread-title {
  font-weight: bold;
  font-size: 1.1rem;
  color: #333;
}

.thread-context {
  font-size: 0.9rem;
  color: #666;
}

.thread-actions {
  display: flex;
  gap: 0.5rem;
}

.leave-thread-btn {
  padding: 0.5rem 1rem;
  background-color: #f44336;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  transition: background-color 0.2s;
}

.leave-thread-btn:hover {
  background-color: #d32f2f;
}
```

### 2. HTML Structure Changes

#### 2.1 Update Main Container Layout

```html
<div class="main-container">
  <div class="rooms-panel">
    <!-- Existing rooms panel content -->
  </div>

  <div class="threads-panel" id="threadsPanel">
    <div class="threads-header">
      <span>Threads</span>
      <button class="create-thread-btn" id="createThreadBtn">
        Create Thread
      </button>
    </div>
    <div class="threads-list" id="threadsList">
      <!-- Threads will be populated here -->
    </div>
  </div>

  <div class="chat-area" id="chatArea">
    <!-- Existing chat area for room messages -->
  </div>

  <div class="thread-view" id="threadView">
    <div class="thread-header">
      <div class="thread-info">
        <div class="thread-title" id="threadTitle">Thread Name</div>
        <div class="thread-context" id="threadContext">In Room Name</div>
      </div>
      <div class="thread-actions">
        <button class="leave-thread-btn" id="leaveThreadBtn">
          Leave Thread
        </button>
      </div>
    </div>
    <div class="messages-container" id="threadMessagesContainer">
      <!-- Thread messages will be displayed here -->
    </div>
    <div class="message-input-container">
      <input
        type="text"
        class="message-input"
        id="threadMessageInput"
        placeholder="Type your thread message..."
        disabled
      />
      <button class="send-btn" id="threadSendBtn" disabled>Send</button>
    </div>
  </div>
</div>
```

#### 2.2 Thread Creation Modal

```html
<!-- Create Thread Modal -->
<div class="modal" id="createThreadModal">
  <div class="modal-content">
    <div class="modal-header">
      <h2>Create New Thread</h2>
    </div>
    <div class="form-group">
      <label for="threadName">Thread Name</label>
      <input type="text" id="threadName" placeholder="Enter thread name" />
    </div>
    <div class="form-group">
      <label for="rootMessageId">Start from message (optional)</label>
      <input type="text" id="rootMessageId" placeholder="Message ID" readonly />
    </div>
    <div class="modal-buttons">
      <button class="btn btn-secondary" id="cancelCreateThread">Cancel</button>
      <button class="btn btn-primary" id="confirmCreateThread">Create</button>
    </div>
  </div>
</div>
```

#### 2.3 Message Thread Indicators

Update message structure to include thread indicators:

```html
<!-- Add to message structure -->
<div class="message-container" data-message-id="${messageId}">
  <div class="message ${isOwn ? "own" : ""}" data-message-id="${messageId}">
    <!-- Existing message content -->

    <!-- Add thread indicator if message has thread -->
    <div class="thread-indicator" id="threadIndicator-${messageId}" style="display: none;">
      🧵 <span class="thread-count">0</span>
    </div>

    <!-- Add thread start badge if this message started a thread -->
    <div class="thread-start-badge" id="threadStart-${messageId}" style="display: none;">
      Thread
    </div>
  </div>
</div>
```

### 3. State Management Variables

Add to the existing state section:

```javascript
// Thread state management
let currentThreadId = null;
let threads = [];
let isViewingThread = false;

// Thread creation state
let creatingThreadFromMessage = null; // { messageId, content }
```

### 4. DOM Elements

Add to the DOM elements section:

```javascript
// Thread-related DOM elements
const threadsPanel = document.getElementById("threadsPanel");
const threadsList = document.getElementById("threadsList");
const createThreadBtn = document.getElementById("createThreadBtn");
const createThreadModal = document.getElementById("createThreadModal");
const cancelCreateThread = document.getElementById("cancelCreateThread");
const confirmCreateThread = document.getElementById("confirmCreateThread");
const threadNameInput = document.getElementById("threadName");
const rootMessageIdInput = document.getElementById("rootMessageId");

// Thread view elements
const threadView = document.getElementById("threadView");
const chatArea = document.getElementById("chatArea");
const threadTitle = document.getElementById("threadTitle");
const threadContext = document.getElementById("threadContext");
const threadMessagesContainer = document.getElementById(
  "threadMessagesContainer"
);
const threadMessageInput = document.getElementById("threadMessageInput");
const threadSendBtn = document.getElementById("threadSendBtn");
const leaveThreadBtn = document.getElementById("leaveThreadBtn");
```

### 5. WebSocket Message Handlers

Add new WebSocket message handlers:

```javascript
// Add to handleWebSocketMessage switch statement
case "new_thread":
  handleNewThread(message.data);
  break;
case "thread_joined":
  handleThreadJoined(message.data);
  break;
case "user_joined_thread":
  handleUserJoinedThread(message.data);
  break;
case "user_left_thread":
  handleUserLeftThread(message.data);
  break;
case "new_thread_message":
  handleNewThreadMessage(message.data);
  break;
```

### 6. Thread Functions

#### 6.1 Thread Management Functions

```javascript
// Load threads for current room
async function loadThreads(roomId) {
  try {
    const response = await fetch(`${API_BASE}/rooms/${roomId}/threads`, {
      headers: {
        "X-Auth-Key": authKey,
        "X-Username": username,
      },
    });

    if (response.ok) {
      threads = await response.json();
      renderThreads();
    } else {
      console.error("Failed to load threads:", response.statusText);
    }
  } catch (error) {
    console.error("Error loading threads:", error);
  }
}

// Render threads list
function renderThreads() {
  threadsList.innerHTML = "";

  threads.forEach((thread) => {
    const threadItem = document.createElement("div");
    threadItem.className = "thread-item";
    threadItem.onclick = () => joinThread(thread.id, thread.name);

    if (thread.id === currentThreadId) {
      threadItem.classList.add("active");
    }

    const createdAt = new Date(thread.createdAt).toLocaleDateString();
    threadItem.innerHTML = `
      <div class="thread-info">
        <div class="thread-name">${thread.name}</div>
        <div class="thread-meta">${thread.memberCount} members • ${createdAt}</div>
      </div>
    `;

    threadsList.appendChild(threadItem);
  });
}

// Create new thread
async function createThread(name, rootMessageId = null) {
  try {
    const response = await fetch(`${API_BASE}/rooms/${currentRoomId}/threads`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-Auth-Key": authKey,
        "X-Username": username,
      },
      body: JSON.stringify({
        name: name,
        rootMessageId: rootMessageId,
      }),
    });

    if (response.ok) {
      const thread = await response.json();
      threads.push(thread);
      renderThreads();
      hideCreateThreadModal();

      // Auto-join the created thread
      joinThread(thread.id, thread.name);
    } else {
      console.error("Failed to create thread:", response.statusText);
      alert("Failed to create thread");
    }
  } catch (error) {
    console.error("Error creating thread:", error);
    alert("Error creating thread");
  }
}

// Join a thread
function joinThread(threadId, threadName) {
  if (currentThreadId === threadId && isViewingThread) {
    return; // Already in this thread
  }

  // Send WebSocket message to join thread
  sendWebSocketMessage("join_thread", { threadId: threadId });
}

// Leave current thread
function leaveThread() {
  if (!currentThreadId) return;

  sendWebSocketMessage("leave_thread", {});

  // Switch back to room view
  showRoomView();
}

// Send message to thread
function sendThreadMessage() {
  const content = threadMessageInput.value.trim();
  if (content && currentThreadId) {
    const messageData = {
      threadId: currentThreadId,
      content: content,
    };

    // Add replyTo if replying
    if (isReplying && replyMessageId) {
      messageData.replyTo = replyMessageId;
    }

    sendWebSocketMessage("send_thread_message", messageData);
    threadMessageInput.value = "";

    // Clear reply state after sending
    if (isReplying) {
      cancelReply();
    }
  }
}
```

#### 6.2 Thread View Management

```javascript
// Switch to thread view
function showThreadView(threadId, threadName) {
  isViewingThread = true;
  currentThreadId = threadId;

  // Update UI
  chatArea.style.display = "none";
  threadView.style.display = "flex";
  threadView.classList.add("active");

  // Update thread info
  threadTitle.textContent = threadName;
  threadContext.textContent = `In ${currentRoomName.textContent}`;

  // Enable thread input
  threadMessageInput.disabled = false;
  threadSendBtn.disabled = false;

  // Load thread messages
  loadThreadMessages(threadId);

  // Update active states
  document.querySelectorAll(".thread-item").forEach((item) => {
    item.classList.remove("active");
  });
  document.querySelector(`[onclick*="${threadId}"]`)?.classList.add("active");
}

// Switch back to room view
function showRoomView() {
  isViewingThread = false;
  currentThreadId = null;

  // Update UI
  threadView.style.display = "none";
  threadView.classList.remove("active");
  chatArea.style.display = "flex";

  // Disable thread input
  threadMessageInput.disabled = true;
  threadSendBtn.disabled = true;

  // Clear thread messages
  threadMessagesContainer.innerHTML = "";

  // Update active states
  document.querySelectorAll(".thread-item").forEach((item) => {
    item.classList.remove("active");
  });
}

// Load messages for thread
async function loadThreadMessages(threadId) {
  try {
    const response = await fetch(`${API_BASE}/threads/${threadId}/messages`, {
      headers: {
        "X-Auth-Key": authKey,
        "X-Username": username,
      },
    });

    if (response.ok) {
      const messages = await response.json();
      threadMessagesContainer.innerHTML = "";
      messages.forEach((message) => {
        if (message.type === "system") {
          addSystemMessage(message.content, threadMessagesContainer);
        } else {
          addThreadMessage(
            message.sender,
            message.content,
            message.timestamp,
            message.sender === username,
            message.id,
            message.reactions || [],
            message.seen || { count: 0, users: [] },
            message.replyTo || null
          );
        }
      });
      scrollToBottom(threadMessagesContainer);
    } else {
      console.error("Failed to load thread messages:", response.statusText);
    }
  } catch (error) {
    console.error("Error loading thread messages:", error);
  }
}
```

#### 6.3 Thread Message Handling

```javascript
// Add message to thread view
function addThreadMessage(
  sender,
  content,
  timestamp,
  isOwn = false,
  messageId = null,
  reactions = [],
  seen = { count: 0, users: [] },
  replyTo = null
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
  messageContainer.appendChild(messageDiv);

  // Add reply button
  const replyBtn = document.createElement("button");
  replyBtn.className = "reply-btn";
  replyBtn.textContent = "Reply";
  replyBtn.onclick = () => startReply(messageId, sender, content);
  messageDiv.appendChild(replyBtn);

  threadMessagesContainer.appendChild(messageContainer);

  // Initialize reactions and seen (reuse existing functions)
  if (reactions && reactions.length > 0) {
    reactions.forEach((reaction) => {
      updateMessageReactions(
        messageId,
        reaction.emoji,
        reaction.count,
        reaction.reactors
      );
    });
  }

  if (seen && (seen.count > 0 || seen.users.length > 0)) {
    updateMessageSeen(messageId, seen.count, seen.users);
  }

  // Add event listeners for reaction picker
  const reactionOptions = messageDiv.querySelectorAll(".reaction-option");
  reactionOptions.forEach((option) => {
    option.addEventListener("click", () => {
      const emoji = option.getAttribute("data-emoji");
      toggleReaction(messageId, emoji);
    });
  });

  scrollToBottom(threadMessagesContainer);
}
```

#### 6.4 WebSocket Event Handlers

```javascript
// Handle new thread event
function handleNewThread(data) {
  const threadData = data;

  // Add to threads list if in current room
  if (threadData.roomId === currentRoomId) {
    threads.push(threadData);
    renderThreads();

    // Update message that started this thread
    if (threadData.rootMessageId) {
      updateMessageThreadIndicator(threadData.rootMessageId, true);
    }
  }
}

// Handle thread joined event
function handleThreadJoined(data) {
  const threadData = data;
  currentThreadId = threadData.threadId;

  // Switch to thread view
  showThreadView(threadData.threadId, threadData.threadName);

  // Add system message
  addSystemMessage(
    `Joined thread: ${threadData.threadName}`,
    threadMessagesContainer
  );
}

// Handle user joined thread event
function handleUserJoinedThread(data) {
  const threadData = data;
  addSystemMessage(
    `${threadData.username} joined the thread`,
    threadMessagesContainer
  );
}

// Handle user left thread event
function handleUserLeftThread(data) {
  const threadData = data;
  addSystemMessage(
    `${threadData.username} left the thread`,
    threadMessagesContainer
  );
}

// Handle new thread message event
function handleNewThreadMessage(data) {
  const messageData = data;

  // Only add if we're in this thread
  if (messageData.threadId === currentThreadId) {
    addThreadMessage(
      messageData.sender,
      messageData.content,
      messageData.timestamp,
      messageData.sender === username,
      messageData.id,
      messageData.reactions || [],
      messageData.seen || { count: 0, users: [] },
      messageData.replyTo || null
    );
  }
}

// Update message thread indicator
function updateMessageThreadIndicator(messageId, hasThread) {
  const indicator = document.getElementById(`threadIndicator-${messageId}`);
  const startBadge = document.getElementById(`threadStart-${messageId}`);

  if (indicator) {
    indicator.style.display = hasThread ? "inline-flex" : "none";
  }

  if (startBadge) {
    startBadge.style.display = hasThread ? "block" : "none";
  }
}
```

### 7. Modal Management

```javascript
// Show/hide create thread modal
function showCreateThreadModal(rootMessageId = null) {
  creatingThreadFromMessage = rootMessageId;

  if (rootMessageId) {
    // Find message content for context
    const messageElement = document.querySelector(
      `[data-message-id="${rootMessageId}"]`
    );
    const messageContent =
      messageElement?.querySelector(".message-content")?.textContent;

    rootMessageIdInput.value = rootMessageId;
    threadNameInput.value = `Thread about: ${messageContent?.substring(
      0,
      50
    )}...`;
    threadNameInput.focus();
    threadNameInput.select();
  } else {
    rootMessageIdInput.value = "";
    threadNameInput.value = "";
    threadNameInput.focus();
  }

  createThreadModal.style.display = "block";
}

function hideCreateThreadModal() {
  createThreadModal.style.display = "none";
  creatingThreadFromMessage = null;
  threadNameInput.value = "";
  rootMessageIdInput.value = "";
}
```

### 8. Event Listeners Update

```javascript
// Add to setupEventListeners function
function setupEventListeners() {
  // Existing event listeners...

  // Thread-related event listeners
  createThreadBtn.addEventListener("click", () => showCreateThreadModal());
  confirmCreateThread.addEventListener("click", createThreadFromModal);
  cancelCreateThread.addEventListener("click", hideCreateThreadModal);
  leaveThreadBtn.addEventListener("click", leaveThread);

  // Thread message input
  threadSendBtn.addEventListener("click", sendThreadMessage);
  threadMessageInput.addEventListener("keypress", (e) => {
    if (e.key === "Enter") {
      sendThreadMessage();
    }
  });

  // Close modal when clicking outside
  window.addEventListener("click", (e) => {
    if (e.target === createThreadModal) {
      hideCreateThreadModal();
    }
  });
}

function createThreadFromModal() {
  const name = threadNameInput.value.trim();
  const rootMessageId = rootMessageIdInput.value.trim();

  if (!name) {
    alert("Thread name is required");
    return;
  }

  createThread(name, rootMessageId || null);
}
```

### 9. Integration with Existing Functions

#### 9.1 Update Room Joined Handler

```javascript
function handleRoomJoined(data) {
  // Existing code...

  // Load threads for this room
  loadThreads(currentRoomId);

  // Show room view by default
  showRoomView();
}
```

#### 9.2 Update Message Display

```javascript
function addMessage(/* existing parameters */) {
  // Existing code...

  // Add thread creation button to messages
  const createThreadBtn = document.createElement("button");
  createThreadBtn.className = "create-thread-from-msg-btn";
  createThreadBtn.textContent = "🧵 Create Thread";
  createThreadBtn.style.cssText = `
    position: absolute;
    top: 0.5rem;
    right: 4rem;
    background-color: transparent;
    border: 1px solid #4a6fa5;
    border-radius: 4px;
    padding: 0.25rem 0.5rem;
    font-size: 0.8rem;
    cursor: pointer;
    color: #4a6fa5;
    opacity: 0;
    transition: opacity 0.2s;
  `;

  createThreadBtn.onclick = () => showCreateThreadModal(messageId);

  messageDiv.appendChild(createThreadBtn);

  // Show on hover
  messageContainer.addEventListener("mouseenter", () => {
    createThreadBtn.style.opacity = "1";
  });

  messageContainer.addEventListener("mouseleave", () => {
    createThreadBtn.style.opacity = "0";
  });

  // Rest of existing code...
}
```

#### 9.3 Update Reply Functions for Thread Context

```javascript
function startReply(messageId, sender, content) {
  // Existing code...

  // Update reply preview to show context (room vs thread)
  if (isViewingThread) {
    messageInput.placeholder = `Replying to ${sender} in thread...`;
  } else {
    messageInput.placeholder = `Replying to ${sender}...`;
  }

  // Rest of existing code...
}
```

### 10. Utility Functions

```javascript
// Updated scrollToBottom to work with any container
function scrollToBottom(container = messagesContainer) {
  container.scrollTop = container.scrollHeight;
}

// Updated addSystemMessage to work with any container
function addSystemMessage(content, container = messagesContainer) {
  const messageDiv = document.createElement("div");
  messageDiv.className = "message system";
  messageDiv.textContent = content;

  container.appendChild(messageDiv);
  scrollToBottom(container);
}
```

## Implementation Order

1. **Add CSS styles** for thread components
2. **Update HTML structure** with thread panels and modals
3. **Add state management** variables
4. **Implement thread management functions** (load, create, join, leave)
5. **Add WebSocket event handlers** for thread events
6. **Update existing functions** to support thread context
7. **Add event listeners** for thread interactions
8. **Test integration** with existing functionality

## Testing Checklist

- [ ] Thread list displays correctly
- [ ] Can create standalone threads
- [ ] Can create threads from messages
- [ ] Can join/leave threads
- [ ] Thread messages display correctly
- [ ] Thread navigation works (room ↔ thread)
- [ ] Reply functionality works in threads
- [ ] Reactions work in threads
- [ ] Seen status works in threads
- [ ] WebSocket events handled correctly
- [ ] UI responsive and usable

## Notes

1. **State Management**: Need to track both room and thread contexts
2. **WebSocket Events**: Thread events are separate from room events
3. **Message Context**: Messages can be in room or thread context
4. **Navigation**: Clear visual indication of current context
5. **Error Handling**: Proper error messages for thread operations
6. **Performance**: Efficient DOM updates for thread lists and messages
