# Simple Chat Client Implementation

This document provides the complete HTML implementation for a simple chat client that connects to the Go WebSocket chat server.

## Overview

The chat client will:

1. Generate a random auth key automatically when the page loads
2. Connect to the WebSocket server with authentication
3. List available rooms and allow joining
4. Send and receive messages in real-time
5. Display connection status

## Complete HTML Code

```html
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Simple Chat Client</title>
    <style>
      * {
        margin: 0;
        padding: 0;
        box-sizing: border-box;
      }

      body {
        font-family: "Segoe UI", Tahoma, Geneva, Verdana, sans-serif;
        background-color: #f5f5f5;
        height: 100vh;
        display: flex;
        flex-direction: column;
      }

      .header {
        background-color: #4a6fa5;
        color: white;
        padding: 1rem;
        display: flex;
        justify-content: space-between;
        align-items: center;
        box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
      }

      .header h1 {
        font-size: 1.5rem;
      }

      .user-info {
        display: flex;
        align-items: center;
        gap: 0.5rem;
      }

      .status-indicator {
        width: 10px;
        height: 10px;
        border-radius: 50%;
        background-color: #ccc;
      }

      .status-indicator.connected {
        background-color: #4caf50;
      }

      .status-indicator.disconnected {
        background-color: #f44336;
      }

      .main-container {
        display: flex;
        flex: 1;
        overflow: hidden;
      }

      .rooms-panel {
        width: 250px;
        background-color: white;
        border-right: 1px solid #ddd;
        display: flex;
        flex-direction: column;
      }

      .rooms-header {
        padding: 1rem;
        border-bottom: 1px solid #eee;
        font-weight: bold;
      }

      .rooms-list {
        flex: 1;
        overflow-y: auto;
      }

      .room-item {
        padding: 0.75rem 1rem;
        cursor: pointer;
        border-bottom: 1px solid #f0f0f0;
        transition: background-color 0.2s;
      }

      .room-item:hover {
        background-color: #f0f0f0;
      }

      .room-item.active {
        background-color: #e3f2fd;
        border-left: 3px solid #4a6fa5;
      }

      .create-room-btn {
        margin: 1rem;
        padding: 0.5rem;
        background-color: #4a6fa5;
        color: white;
        border: none;
        border-radius: 4px;
        cursor: pointer;
        transition: background-color 0.2s;
      }

      .create-room-btn:hover {
        background-color: #3a5a8c;
      }

      .chat-area {
        flex: 1;
        display: flex;
        flex-direction: column;
        background-color: white;
      }

      .chat-header {
        padding: 1rem;
        border-bottom: 1px solid #eee;
        background-color: #fafafa;
      }

      .messages-container {
        flex: 1;
        overflow-y: auto;
        padding: 1rem;
      }

      .message {
        margin-bottom: 1rem;
        padding: 0.5rem;
        border-radius: 8px;
        max-width: 80%;
      }

      .message.own {
        background-color: #e3f2fd;
        margin-left: auto;
        text-align: right;
      }

      .message.system {
        background-color: #f5f5f5;
        text-align: center;
        font-style: italic;
        color: #666;
        max-width: 100%;
      }

      .message-info {
        font-size: 0.8rem;
        color: #666;
        margin-bottom: 0.25rem;
      }

      .message-content {
        word-wrap: break-word;
      }

      .message-input-container {
        display: flex;
        padding: 1rem;
        border-top: 1px solid #eee;
        background-color: #fafafa;
      }

      .message-input {
        flex: 1;
        padding: 0.75rem;
        border: 1px solid #ddd;
        border-radius: 4px;
        font-size: 1rem;
      }

      .send-btn {
        margin-left: 0.5rem;
        padding: 0.75rem 1.5rem;
        background-color: #4a6fa5;
        color: white;
        border: none;
        border-radius: 4px;
        cursor: pointer;
        transition: background-color 0.2s;
      }

      .send-btn:hover:not(:disabled) {
        background-color: #3a5a8c;
      }

      .send-btn:disabled {
        background-color: #ccc;
        cursor: not-allowed;
      }

      .connection-status {
        padding: 0.5rem 1rem;
        text-align: center;
        font-size: 0.9rem;
      }

      .connection-status.connected {
        color: #4caf50;
      }

      .connection-status.disconnected {
        color: #f44336;
      }

      .modal {
        display: none;
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background-color: rgba(0, 0, 0, 0.5);
        z-index: 1000;
      }

      .modal-content {
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

      .modal-header {
        margin-bottom: 1rem;
      }

      .form-group {
        margin-bottom: 1rem;
      }

      .form-group label {
        display: block;
        margin-bottom: 0.5rem;
      }

      .form-group input {
        width: 100%;
        padding: 0.5rem;
        border: 1px solid #ddd;
        border-radius: 4px;
      }

      .modal-buttons {
        display: flex;
        justify-content: flex-end;
        gap: 0.5rem;
      }

      .btn {
        padding: 0.5rem 1rem;
        border: none;
        border-radius: 4px;
        cursor: pointer;
      }

      .btn-primary {
        background-color: #4a6fa5;
        color: white;
      }

      .btn-secondary {
        background-color: #ddd;
      }
    </style>
  </head>
  <body>
    <header class="header">
      <h1>Simple Chat Client</h1>
      <div class="user-info">
        <span id="username">Loading...</span>
        <div class="status-indicator" id="statusIndicator"></div>
      </div>
    </header>

    <div class="main-container">
      <div class="rooms-panel">
        <div class="rooms-header">Rooms</div>
        <div class="rooms-list" id="roomsList">
          <!-- Rooms will be populated here -->
        </div>
        <button class="create-room-btn" id="createRoomBtn">
          Create New Room
        </button>
      </div>

      <div class="chat-area">
        <div class="chat-header">
          <div id="currentRoomName">Select a room to start chatting</div>
        </div>
        <div class="messages-container" id="messagesContainer">
          <!-- Messages will be displayed here -->
        </div>
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
    </div>

    <div class="connection-status" id="connectionStatus">
      Connecting to server...
    </div>

    <!-- Create Room Modal -->
    <div class="modal" id="createRoomModal">
      <div class="modal-content">
        <div class="modal-header">
          <h2>Create New Room</h2>
        </div>
        <div class="form-group">
          <label for="roomName">Room Name</label>
          <input type="text" id="roomName" placeholder="Enter room name" />
        </div>
        <div class="form-group">
          <label for="roomDescription">Description (optional)</label>
          <input
            type="text"
            id="roomDescription"
            placeholder="Enter room description"
          />
        </div>
        <div class="modal-buttons">
          <button class="btn btn-secondary" id="cancelCreateRoom">
            Cancel
          </button>
          <button class="btn btn-primary" id="confirmCreateRoom">Create</button>
        </div>
      </div>
    </div>

    <script>
      // Configuration
      const API_BASE = "http://localhost:8080/api";
      const WS_URL = "ws://localhost:8080/ws";

      // State
      let authKey = "";
      let username = "";
      let ws = null;
      let currentRoomId = null;
      let rooms = [];

      // DOM Elements
      const usernameEl = document.getElementById("username");
      const statusIndicator = document.getElementById("statusIndicator");
      const connectionStatus = document.getElementById("connectionStatus");
      const roomsList = document.getElementById("roomsList");
      const currentRoomName = document.getElementById("currentRoomName");
      const messagesContainer = document.getElementById("messagesContainer");
      const messageInput = document.getElementById("messageInput");
      const sendBtn = document.getElementById("sendBtn");
      const createRoomBtn = document.getElementById("createRoomBtn");
      const createRoomModal = document.getElementById("createRoomModal");
      const cancelCreateRoom = document.getElementById("cancelCreateRoom");
      const confirmCreateRoom = document.getElementById("confirmCreateRoom");
      const roomNameInput = document.getElementById("roomName");
      const roomDescriptionInput = document.getElementById("roomDescription");

      // Initialize
      function init() {
        generateAuthKey();
        connectWebSocket();
        loadRooms();
        setupEventListeners();
      }

      // Generate random auth key and username
      function generateAuthKey() {
        authKey =
          "auth-" +
          Math.random().toString(36).substring(2, 15) +
          Math.random().toString(36).substring(2, 15);
        username = "user-" + Math.random().toString(36).substring(2, 8);
        usernameEl.textContent = username;
      }

      // Connect to WebSocket
      function connectWebSocket() {
        updateConnectionStatus("connecting");

        ws = new WebSocket(WS_URL);

        ws.onopen = function () {
          updateConnectionStatus("connected");
          console.log("WebSocket connected");
        };

        ws.onmessage = function (event) {
          const message = JSON.parse(event.data);
          handleWebSocketMessage(message);
        };

        ws.onclose = function () {
          updateConnectionStatus("disconnected");
          console.log("WebSocket disconnected");
          // Try to reconnect after 3 seconds
          setTimeout(connectWebSocket, 3000);
        };

        ws.onerror = function (error) {
          console.error("WebSocket error:", error);
          updateConnectionStatus("disconnected");
        };
      }

      // Handle WebSocket messages
      function handleWebSocketMessage(message) {
        console.log("Received WebSocket message:", message);

        switch (message.event) {
          case "room_joined":
            handleRoomJoined(message.data);
            break;
          case "user_joined":
            handleUserJoined(message.data);
            break;
          case "user_left":
            handleUserLeft(message.data);
            break;
          case "new_message":
            handleNewMessage(message.data);
            break;
          default:
            console.log("Unknown message event:", message.event);
        }
      }

      // Handle room joined event
      function handleRoomJoined(data) {
        const roomData = JSON.parse(data);
        console.log("Joined room:", roomData);
        currentRoomId = roomData.roomId;
        currentRoomName.textContent = roomData.roomName;
        messageInput.disabled = false;
        sendBtn.disabled = false;
        loadMessages(currentRoomId);
      }

      // Handle user joined event
      function handleUserJoined(data) {
        const userData = JSON.parse(data);
        addSystemMessage(`${userData.username} joined the room`);
      }

      // Handle user left event
      function handleUserLeft(data) {
        const userData = JSON.parse(data);
        addSystemMessage(`${userData.username} left the room`);
      }

      // Handle new message event
      function handleNewMessage(data) {
        const messageData = JSON.parse(data);
        addMessage(
          messageData.sender,
          messageData.content,
          messageData.timestamp,
          messageData.sender === username
        );
      }

      // Send WebSocket message
      function sendWebSocketMessage(event, data) {
        if (ws && ws.readyState === WebSocket.OPEN) {
          const message = {
            event: event,
            data: JSON.stringify(data),
          };
          ws.send(JSON.stringify(message));
        } else {
          console.error("WebSocket is not connected");
        }
      }

      // Load rooms from API
      async function loadRooms() {
        try {
          const response = await fetch(`${API_BASE}/rooms`, {
            headers: {
              "X-Auth-Key": authKey,
              "X-Username": username,
            },
          });

          if (response.ok) {
            rooms = await response.json();
            renderRooms();
          } else {
            console.error("Failed to load rooms:", response.statusText);
          }
        } catch (error) {
          console.error("Error loading rooms:", error);
        }
      }

      // Render rooms list
      function renderRooms() {
        roomsList.innerHTML = "";

        rooms.forEach((room) => {
          const roomItem = document.createElement("div");
          roomItem.className = "room-item";
          roomItem.textContent = `${room.name} (${room.memberCount})`;
          roomItem.onclick = () => joinRoom(room.id, room.name);

          if (room.id === currentRoomId) {
            roomItem.classList.add("active");
          }

          roomsList.appendChild(roomItem);
        });
      }

      // Join a room
      function joinRoom(roomId, roomName) {
        if (currentRoomId === roomId) {
          return; // Already in this room
        }

        // Leave current room if any
        if (currentRoomId) {
          sendWebSocketMessage("leave_room", {});
        }

        // Join new room
        sendWebSocketMessage("join_room", { roomId: roomId });

        // Update UI
        document.querySelectorAll(".room-item").forEach((item) => {
          item.classList.remove("active");
        });
        event.target.classList.add("active");
      }

      // Load messages for current room
      async function loadMessages(roomId) {
        try {
          const response = await fetch(`${API_BASE}/rooms/${roomId}/messages`, {
            headers: {
              "X-Auth-Key": authKey,
              "X-Username": username,
            },
          });

          if (response.ok) {
            const messages = await response.json();
            messagesContainer.innerHTML = "";
            messages.forEach((message) => {
              if (message.type === "system") {
                addSystemMessage(message.content);
              } else {
                addMessage(
                  message.sender,
                  message.content,
                  message.timestamp,
                  message.sender === username
                );
              }
            });
            scrollToBottom();
          } else {
            console.error("Failed to load messages:", response.statusText);
          }
        } catch (error) {
          console.error("Error loading messages:", error);
        }
      }

      // Add message to chat
      function addMessage(sender, content, timestamp, isOwn = false) {
        const messageDiv = document.createElement("div");
        messageDiv.className = `message ${isOwn ? "own" : ""}`;

        const time = new Date(timestamp).toLocaleTimeString();

        messageDiv.innerHTML = `
                <div class="message-info">${sender} • ${time}</div>
                <div class="message-content">${escapeHtml(content)}</div>
            `;

        messagesContainer.appendChild(messageDiv);
        scrollToBottom();
      }

      // Add system message
      function addSystemMessage(content) {
        const messageDiv = document.createElement("div");
        messageDiv.className = "message system";
        messageDiv.textContent = content;

        messagesContainer.appendChild(messageDiv);
        scrollToBottom();
      }

      // Send message
      function sendMessage() {
        const content = messageInput.value.trim();
        if (content && currentRoomId) {
          sendWebSocketMessage("send_message", {
            roomId: currentRoomId,
            content: content,
          });
          messageInput.value = "";
        }
      }

      // Create new room
      async function createRoom() {
        const name = roomNameInput.value.trim();
        const description = roomDescriptionInput.value.trim();

        if (!name) {
          alert("Room name is required");
          return;
        }

        try {
          const response = await fetch(`${API_BASE}/rooms`, {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
              "X-Auth-Key": authKey,
              "X-Username": username,
            },
            body: JSON.stringify({
              name: name,
              description: description,
            }),
          });

          if (response.ok) {
            const room = await response.json();
            rooms.push(room);
            renderRooms();
            hideCreateRoomModal();
            roomNameInput.value = "";
            roomDescriptionInput.value = "";
          } else {
            console.error("Failed to create room:", response.statusText);
            alert("Failed to create room");
          }
        } catch (error) {
          console.error("Error creating room:", error);
          alert("Error creating room");
        }
      }

      // Update connection status
      function updateConnectionStatus(status) {
        statusIndicator.className = `status-indicator ${status}`;
        connectionStatus.className = `connection-status ${status}`;

        switch (status) {
          case "connected":
            connectionStatus.textContent = "Connected to server";
            break;
          case "disconnected":
            connectionStatus.textContent = "Disconnected from server";
            break;
          case "connecting":
            connectionStatus.textContent = "Connecting to server...";
            break;
        }
      }

      // Show/hide create room modal
      function showCreateRoomModal() {
        createRoomModal.style.display = "block";
      }

      function hideCreateRoomModal() {
        createRoomModal.style.display = "none";
      }

      // Utility functions
      function escapeHtml(text) {
        const div = document.createElement("div");
        div.textContent = text;
        return div.innerHTML;
      }

      function scrollToBottom() {
        messagesContainer.scrollTop = messagesContainer.scrollHeight;
      }

      // Setup event listeners
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

        // Close modal when clicking outside
        window.addEventListener("click", (e) => {
          if (e.target === createRoomModal) {
            hideCreateRoomModal();
          }
        });
      }

      // Initialize the app when DOM is loaded
      document.addEventListener("DOMContentLoaded", init);
    </script>
  </body>
</html>
```

## How to Use

1. Save the code above as `chat.html` in the same directory as your Go server
2. Start the Go server (`go run main.go`)
3. Open `chat.html` in a web browser
4. The client will automatically generate a random auth key and username
5. Select a room from the list to start chatting
6. Type your message and press Enter or click Send

## Features

- **Automatic Authentication**: Generates a random auth key and username on page load
- **Real-time Messaging**: Uses WebSocket for instant message delivery
- **Room Management**: List available rooms and join them
- **Create Rooms**: Create new chat rooms with name and description
- **Message History**: Loads previous messages when joining a room
- **Connection Status**: Shows real-time connection status
- **Responsive Design**: Clean and modern interface that works on different screen sizes
- **System Messages**: Shows when users join or leave rooms

## Technical Details

- **WebSocket Events**: Handles all server events (room_joined, user_joined, user_left, new_message)
- **API Integration**: Uses REST API for room management and message history
- **Error Handling**: Gracefully handles connection issues and API errors
- **Auto-reconnection**: Automatically reconnects if WebSocket connection is lost
- **Security**: Escapes HTML content to prevent XSS attacks

This implementation provides a complete, functional chat client that works seamlessly with your Go WebSocket server.
