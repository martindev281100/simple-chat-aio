# Thread Implementation Guide for chat.html

This guide provides the exact code changes needed to implement thread support in chat.html, following the implementation plan.

## Step 1: Add Thread CSS Styles

Add the following CSS styles to the existing `<style>` section in chat.html (after line 462):

```css
/* Thread Styles */
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
  padding: 0.5rem;
  background-color: #4a6fa5;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  transition: background-color 0.2s;
  font-size: 0.8rem;
}

.create-thread-btn:hover {
  background-color: #3a5a8c;
}

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

.create-thread-from-msg-btn {
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
}

.create-thread-from-msg-btn:hover {
  background-color: #f0f0f0;
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

## Step 2: Update HTML Structure

### 2.1 Update Main Container Layout

Replace the existing `.main-container` div (lines 474-504) with:

```html
<div class="main-container">
  <div class="rooms-panel">
    <div class="rooms-header">Rooms</div>
    <div class="rooms-list" id="roomsList">
      <!-- Rooms will be populated here -->
    </div>
    <button class="create-room-btn" id="createRoomBtn">Create New Room</button>
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
    <div class="chat-header">
      <div id="currentRoomName">Select a room to start chatting</div>
    </div>
    <div class="messages-container" id="messagesContainer">
      <!-- Messages will be displayed here -->
    </div>
    <div id="replyPreviewContainer"></div>
    <div class="message-input-container">
      <input
        type="text"
        class="message-input"
        id="messageInput"
        placeholder="Type your message..."
        disabled
      />
      <button class="send-btn" id="sendBtn" disabled>Send</button>
    </div>
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
    <div id="threadReplyPreviewContainer"></div>
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

### 2.2 Add Thread Creation Modal

Add the thread creation modal after the existing create room modal (after line 535):

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

## Step 3: Add Thread State Management

Add the following state variables after line 551:

```javascript
// Thread state management
let currentThreadId = null;
let threads = [];
let isViewingThread = false;

// Thread creation state
let creatingThreadFromMessage = null; // { messageId, content }
```

## Step 4: Add Thread DOM Elements

Add the following DOM element declarations after line 580:

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
const threadReplyPreviewContainer = document.getElementById(
  "threadReplyPreviewContainer"
);
```

## Step 5: Update WebSocket Message Handlers

Add the following cases to the `handleWebSocketMessage` function after line 737:

```javascript
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

## Step 6: Add Thread Management Functions

Add the following functions after the `cancelReply` function (after line 1364):

```javascript
// Thread Management Functions
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

function joinThread(threadId, threadName) {
  if (currentThreadId === threadId && isViewingThread) {
    return; // Already in this thread
  }

  // Send WebSocket message to join thread
  sendWebSocketMessage("join_thread", { threadId: threadId });
}

function leaveThread() {
  if (!currentThreadId) return;

  sendWebSocketMessage("leave_thread", {});

  // Switch back to room view
  showRoomView();
}

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

## Step 7: Add Thread View Management

Add the following functions after the thread management functions:

```javascript
// Thread View Management
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

## Step 8: Add Thread Message Handling

Add the following functions after the thread view management functions:

```javascript
// Thread Message Handling
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

## Step 9: Add WebSocket Event Handlers

Add the following functions after the thread message handling functions:

```javascript
// WebSocket Event Handlers for Threads
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

function handleUserJoinedThread(data) {
  const threadData = data;
  addSystemMessage(
    `${threadData.username} joined the thread`,
    threadMessagesContainer
  );
}

function handleUserLeftThread(data) {
  const threadData = data;
  addSystemMessage(
    `${threadData.username} left the thread`,
    threadMessagesContainer
  );
}

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

## Step 10: Add Modal Management

Add the following functions after the WebSocket event handlers:

```javascript
// Modal Management
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

## Step 11: Update Utility Functions

Update the following utility functions:

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

## Step 12: Update Existing Functions

### 12.1 Update handleRoomJoined Function

Update the `handleRoomJoined` function (around line 744) to include thread loading:

```javascript
function handleRoomJoined(data) {
  console.log("Raw data received:", data);
  console.log("Type of data:", typeof data);
  const roomData = data; // Data is already parsed by WebSocket
  console.log("Room data:", roomData);
  currentRoomId = roomData.roomId;
  currentRoomName.textContent = roomData.roomName;
  messageInput.disabled = false;
  sendBtn.disabled = false;
  isConnectingToRoom = false;
  pendingRoomId = null;
  pendingRoomName = null;
  loadMessages(currentRoomId);

  // Load threads for this room
  loadThreads(currentRoomId);

  // Show room view by default
  showRoomView();

  // Reconnect seen observer after joining room
  if (seenObserver) {
    // Observe all existing messages
    document
      .querySelectorAll(".message[data-message-id]")
      .forEach((messageEl) => {
        const messageId = messageEl.getAttribute("data-message-id");
        const isOwn = messageEl.classList.contains("own");
        if (messageId && !isOwn) {
          seenObserver.observe(messageEl);
        }
      });
  }
}
```

### 12.2 Update addMessage Function

Update the `addMessage` function (around line 977) to include thread creation button:

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

  messageContainer.appendChild(messageDiv);

  // Add reply button
  const replyBtn = document.createElement("button");
  replyBtn.className = "reply-btn";
  replyBtn.textContent = "Reply";
  replyBtn.onclick = () => startReply(messageId, sender, content);
  messageDiv.appendChild(replyBtn);

  // Add thread creation button
  const createThreadBtn = document.createElement("button");
  createThreadBtn.className = "create-thread-from-msg-btn";
  createThreadBtn.textContent = "🧵 Create Thread";
  createThreadBtn.onclick = () => showCreateThreadModal(messageId);
  messageDiv.appendChild(createThreadBtn);

  // Add thread indicator
  const threadIndicator = document.createElement("div");
  threadIndicator.className = "thread-indicator";
  threadIndicator.id = `threadIndicator-${messageId}`;
  threadIndicator.style.display = "none";
  threadIndicator.innerHTML = `🧵 <span class="thread-count">0</span>`;
  messageDiv.appendChild(threadIndicator);

  messagesContainer.appendChild(messageContainer);

  // Show/hide thread button on hover
  messageContainer.addEventListener("mouseenter", () => {
    createThreadBtn.style.opacity = "1";
  });

  messageContainer.addEventListener("mouseleave", () => {
    createThreadBtn.style.opacity = "0";
  });

  // Initialize reactions if provided
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

  // Initialize seen information if provided
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

  // Start observing new message for seen tracking
  if (seenObserver && messageId && !isOwn) {
    seenObserver.observe(messageDiv);
  }

  scrollToBottom();
}
```

### 12.3 Update setupEventListeners Function

Update the `setupEventListeners` function (around line 1377) to include thread event listeners:

```javascript
function setupEventListeners() {
  // Send message
  sendBtn.addEventListener("click", sendMessage);
  messageInput.addEventListener("keypress", (e) => {
    if (e.key === "Enter") {
      sendMessage();
    }
  });

  // Create room
  createRoomBtn.addEventListener("click", showCreateRoomModal);
  confirmCreateRoom.addEventListener("click", createRoom);
  cancelCreateRoom.addEventListener("click", hideCreateRoomModal);

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
    if (e.target === createRoomModal) {
      hideCreateRoomModal();
    }
    if (e.target === createThreadModal) {
      hideCreateThreadModal();
    }
  });
}
```

## Step 13: Update Reply Functions

Update the `startReply` function (around line 1302) to handle thread context:

```javascript
function startReply(messageId, sender, content) {
  replyingTo = { messageId, sender, content };
  replyMessageId = messageId;
  isReplying = true;

  // Update message input placeholder based on context
  if (isViewingThread) {
    messageInput.placeholder = `Replying to ${sender} in thread...`;
  } else {
    messageInput.placeholder = `Replying to ${sender}...`;
  }

  // Show reply preview in appropriate input area
  if (isViewingThread) {
    showThreadReplyPreview();
  } else {
    showReplyPreview();
  }

  // Focus on appropriate input
  if (isViewingThread) {
    threadMessageInput.focus();
  } else {
    messageInput.focus();
  }
}
```

Add the thread reply preview function:

```javascript
function showThreadReplyPreview() {
  if (!isReplying || !replyingTo) return;

  // Remove existing reply preview if any
  const existingPreview = document.querySelector(".thread-reply-input-preview");
  if (existingPreview) {
    existingPreview.remove();
  }

  // Create reply preview element
  const replyPreview = document.createElement("div");
  replyPreview.className = "reply-input-preview thread-reply-input-preview";
  replyPreview.innerHTML = `
    <div class="reply-preview">
      <span class="reply-preview-sender">Replying to ${replyingTo.sender}</span>
      <div class="reply-preview-content">${escapeHtml(replyingTo.content)}</div>
      <button class="reply-cancel-btn" onclick="cancelReply()">✕</button>
    </div>
  `;

  // Insert before thread message input container
  const inputContainer = threadView.querySelector(".message-input-container");
  inputContainer.parentNode.insertBefore(replyPreview, inputContainer);
}
```

## Final Implementation Notes

1. **File Structure**: All changes should be made to the existing `chat.html` file
2. **Order of Operations**: Follow the steps in order for best results
3. **Testing**: Test each feature incrementally as you implement
4. **Error Handling**: The implementation includes basic error handling
5. **Browser Compatibility**: Uses modern JavaScript features (ES6+)

## Testing Checklist After Implementation

- [ ] Thread panel appears and shows threads
- [ ] Can create standalone threads
- [ ] Can create threads from messages
- [ ] Can join/leave threads
- [ ] Thread messages display correctly
- [ ] Navigation between room and thread views works
- [ ] Reply functionality works in both contexts
- [ ] Reactions work in threads
- [ ] Seen status works in threads
- [ ] WebSocket events are handled correctly
- [ ] UI is responsive and usable

This comprehensive guide provides all the code changes needed to implement full thread support in chat.html, matching the backend functionality implemented in main.go.
