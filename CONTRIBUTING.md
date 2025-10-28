# Contributing to Simple Chat AIO

Thank you for your interest in contributing to Simple Chat AIO! This document provides guidelines and information for contributors.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Coding Standards](#coding-standards)
- [Testing Guidelines](#testing-guidelines)
- [Documentation](#documentation)
- [Submitting Changes](#submitting-changes)
- [Review Process](#review-process)

## Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](https://www.contributor-covenant.org/version/2/0/code_of_conduct/). Please read and follow these guidelines to ensure a welcoming environment for all contributors.

## Getting Started

### Prerequisites

- Go 1.25.1 or higher
- Git
- Basic knowledge of WebSocket and real-time applications
- Familiarity with HTML/CSS/JavaScript for frontend contributions

### Initial Setup

1. Fork the repository on GitHub
2. Clone your fork locally:

```bash
git clone https://github.com/YOUR_USERNAME/simple-chat-aio.git
cd simple-chat-aio
```

3. Add the original repository as upstream:

```bash
git remote add upstream https://github.com/ORIGINAL_OWNER/simple-chat-aio.git
```

4. Install dependencies:

```bash
go mod download
```

5. Verify the setup:

```bash
go run main.go
```

## Development Workflow

### 1. Create a Branch

Create a new branch for your feature or bugfix:

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/bug-description
```

Branch naming conventions:

- `feature/` - New features
- `fix/` - Bug fixes
- `docs/` - Documentation changes
- `refactor/` - Code refactoring
- `test/` - Test additions or improvements

### 2. Make Changes

- Make small, focused changes
- Follow the coding standards outlined below
- Test your changes thoroughly
- Update documentation as needed

### 3. Test Your Changes

Run the application and test your changes:

```bash
go run main.go
```

Test in multiple browsers if making frontend changes:

- Chrome/Chromium
- Firefox
- Safari (if available)

### 4. Commit Your Changes

Write clear, descriptive commit messages:

```bash
git add .
git commit -m "feat: add thread creation functionality"
```

Commit message format:

```
type(scope): description

[optional body]

[optional footer]
```

Types:

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Test additions
- `chore`: Maintenance tasks

### 5. Push and Create Pull Request

Push your branch to your fork:

```bash
git push origin feature/your-feature-name
```

Create a pull request on GitHub with:

- Clear title and description
- Reference any relevant issues
- Include screenshots for UI changes
- Describe testing performed

## Coding Standards

### Go Backend

#### Formatting

- Use `gofmt` for code formatting
- Use `golint` for linting
- Maximum line length: 120 characters

#### Naming Conventions

```go
// Packages: lowercase, single word
package chat

// Constants: UPPER_SNAKE_CASE
const MAX_MESSAGE_LENGTH = 1000

// Variables: camelCase
var messageCount int

// Functions: camelCase
func sendMessage(roomID string, content string) error

// Structs: PascalCase
type Message struct {
    ID        string    `json:"id"`
    Content   string    `json:"content"`
    Timestamp time.Time `json:"timestamp"`
}

// Interfaces: PascalCase, often ending in "er"
type MessageSender interface {
    Send(msg Message) error
}
```

#### Error Handling

- Always handle errors explicitly
- Use descriptive error messages
- Wrap errors when adding context:

```go
if err != nil {
    return fmt.Errorf("failed to send message: %w", err)
}
```

#### Comments

- Comment public functions and exported types
- Use godoc format:

```go
// SendMessage sends a message to the specified room.
// It returns an error if the message cannot be delivered.
func SendMessage(roomID, content string) error {
    // implementation
}
```

### JavaScript Frontend

#### Formatting

- Use 2 spaces for indentation
- Use semicolons consistently
- Maximum line length: 100 characters

#### Naming Conventions

```javascript
// Variables and functions: camelCase
let messageCount = 0;
function sendMessage(roomId, content) {}

// Constants: UPPER_SNAKE_CASE
const MAX_MESSAGE_LENGTH = 1000;

// Classes: PascalCase
class MessageHandler {}

// DOM elements: kebab-case with element prefix
const messageInput = document.getElementById("message-input");
const sendButton = document.getElementById("send-button");
```

#### Event Handling

- Use event delegation for dynamic content
- Remove event listeners when no longer needed
- Use passive event listeners when possible:

```javascript
// Good
document.addEventListener("click", handleClick, { passive: true });

// Avoid memory leaks
element.removeEventListener("click", handleClick);
```

#### Error Handling

- Always handle promise rejections
- Provide user-friendly error messages
- Log detailed errors for debugging:

```javascript
try {
  await sendMessage(message);
} catch (error) {
  console.error("Failed to send message:", error);
  showUserError("Message could not be sent. Please try again.");
}
```

### WebSocket Implementation

When working with WebSocket functionality:

1. **Connection Management**

   - Handle connection failures gracefully
   - Implement reconnection logic
   - Clean up connections on page unload

2. **Message Format**

   - Use consistent JSON structure
   - Include event type and data
   - Validate message format before sending

3. **Error Handling**
   - Handle WebSocket errors
   - Implement fallback for unsupported browsers
   - Show connection status to users

Example WebSocket event handling:

```javascript
function handleWebSocketMessage(event) {
  try {
    const message = JSON.parse(event.data);

    if (!message.event || !message.data) {
      console.warn("Invalid message format:", message);
      return;
    }

    switch (message.event) {
      case "new_message":
        handleNewMessage(message.data);
        break;
      // ... other cases
    }
  } catch (error) {
    console.error("Failed to parse WebSocket message:", error);
  }
}
```

## Testing Guidelines

### Unit Testing

Write tests for new functions and features:

```go
func TestSendMessage(t *testing.T) {
    // Setup
    room := createTestRoom()
    message := "Hello, world!"

    // Execute
    err := SendMessage(room.ID, message)

    // Assert
    assert.NoError(t, err)
    assert.Contains(t, room.Messages, message)
}
```

### Integration Testing

Test WebSocket communication:

```go
func TestWebSocketMessageFlow(t *testing.T) {
    // Setup test server and client
    server := setupTestServer()
    client := setupTestClient(server.URL)

    // Test message flow
    client.sendJSON(map[string]interface{}{
        "event": "join_room",
        "data": map[string]string{"roomId": "test-room"},
    })

    // Verify server received and processed message
    assert.Eventually(t, func() bool {
        return server.receivedJoinEvent()
    }, 5*time.Second, 100*time.Millisecond)
}
```

### Frontend Testing

Test JavaScript functionality:

```javascript
// Example using Jest or similar framework
describe("Message Sending", () => {
  test("should send message when send button clicked", () => {
    // Setup DOM
    document.body.innerHTML = `
            <input id="message-input" value="Test message">
            <button id="send-button">Send</button>
        `;

    // Mock WebSocket
    const mockWs = { send: jest.fn() };
    global.WebSocket = jest.fn(() => mockWs);

    // Initialize and test
    initializeChat();
    document.getElementById("send-button").click();

    // Assert
    expect(mockWs.send).toHaveBeenCalledWith(
      expect.stringContaining("Test message")
    );
  });
});
```

### Manual Testing

For WebSocket features, always perform manual testing:

1. **Connection Testing**

   - Test connection establishment
   - Test reconnection after disconnection
   - Test with different browsers

2. **Real-time Features**

   - Test message delivery speed
   - Test concurrent users
   - Test reaction updates
   - Test seen status updates

3. **Error Scenarios**
   - Test network interruptions
   - Test invalid message formats
   - Test server restarts

## Documentation

### Code Documentation

- Document all public functions and types
- Use godoc format for Go code
- Include usage examples for complex functions

### README Updates

When adding features:

1. Update the features list
2. Add new API endpoints to documentation
3. Include WebSocket events if applicable
4. Update configuration options if changed

### Inline Comments

Add comments for:

- Complex algorithms
- Business logic decisions
- Temporary workarounds
- Performance considerations

## Submitting Changes

### Pull Request Checklist

Before submitting a pull request, ensure:

- [ ] Code follows project coding standards
- [ ] All tests pass
- [ ] Documentation is updated
- [ ] Commit messages are clear and descriptive
- [ ] No sensitive information is included
- [ ] WebSocket functionality is tested (if applicable)
- [ ] Frontend works in multiple browsers

### Pull Request Template

Use this template for your pull request:

```markdown
## Description

Brief description of changes and motivation.

## Type of Change

- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing

Describe how you tested your changes:

- Unit tests
- Integration tests
- Manual testing in browsers
- WebSocket connection testing

## Checklist

- [ ] My code follows the style guidelines
- [ ] I have performed a self-review
- [ ] I have commented my code where necessary
- [ ] I have updated documentation accordingly
- [ ] My changes generate no new warnings
```

## Review Process

### Review Criteria

Pull requests are evaluated based on:

1. **Functionality**

   - Does the code work as intended?
   - Are there any regressions?
   - Is the implementation complete?

2. **Code Quality**

   - Is the code readable and maintainable?
   - Are there appropriate comments?
   - Is error handling adequate?

3. **Performance**

   - Does the change impact performance?
   - Are there memory leaks?
   - Is WebSocket efficiency maintained?

4. **Testing**
   - Are tests comprehensive?
   - Do tests cover edge cases?
   - Is manual testing documented?

### Review Process

1. **Initial Review**

   - Automated checks pass
   - Code style compliance
   - Basic functionality verification

2. **Detailed Review**

   - Architecture and design
   - Security implications
   - Performance impact

3. **Testing Review**

   - Test coverage analysis
   - Manual testing verification
   - Cross-browser compatibility

4. **Approval and Merge**
   - Address all review comments
   - Update documentation as needed
   - Merge to main branch

## Getting Help

If you need help with contributing:

1. Check existing [issues](https://github.com/ORIGINAL_OWNER/simple-chat-aio/issues)
2. Create a new issue with the `question` label
3. Join discussions in existing issues
4. Refer to [documentation](docs/) for technical details

## Recognition

Contributors are recognized in:

- README.md contributors section
- Release notes for significant contributions
- Commit history attribution

Thank you for contributing to Simple Chat AIO!
