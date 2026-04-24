# Chat History Storage Design

## Overview
Store AI chat conversation history in MySQL instead of localStorage.

## Data Model

### conversations table
| Field | Type | Description |
|-------|------|-------------|
| id | BIGINT | Primary key, auto increment |
| user_id | BIGINT | User ID |
| title | VARCHAR(255) | Conversation title |
| created_at | DATETIME | Creation time |
| updated_at | DATETIME | Last update time |

### chat_messages table
| Field | Type | Description |
|-------|------|-------------|
| id | BIGINT | Primary key, auto increment |
| conversation_id | BIGINT | Foreign key to conversations |
| role | VARCHAR(20) | "user" or "assistant" |
| content | TEXT | Message content |
| created_at | DATETIME | Creation time |

## API Design

### List Conversations
```
GET /api/chat/conversations
```
Response:
```json
{
  "conversations": [
    {
      "id": 1,
      "title": "New Conversation",
      "updated_at": "2026-04-08T10:00:00Z"
    }
  ]
}
```

### Create Conversation
```
POST /api/chat/conversations
```
Request:
```json
{
  "title": "New Conversation"
}
```

### Get Conversation with Messages
```
GET /api/chat/conversations/:id
```
Response:
```json
{
  "id": 1,
  "title": "My Conversation",
  "messages": [
    {"id": 1, "role": "user", "content": "Hello", "created_at": "..."},
    {"id": 2, "role": "assistant", "content": "Hi!", "created_at": "..."}
  ]
}
```

### Delete Conversation
```
DELETE /api/chat/conversations/:id
```

### Send Message
```
POST /api/chat/messages
```
Request:
```json
{
  "conversation_id": 1,
  "content": "Hello AI"
}
```
Response:
```json
{
  "message": {
    "id": 3,
    "role": "user",
    "content": "Hello AI",
    "created_at": "..."
  },
  "reply": {
    "id": 4,
    "role": "assistant",
    "content": "Response from AI",
    "created_at": "..."
  }
}
```

## Implementation Steps

1. Create model: conversation.go, chat_message.go
2. Create repository: conversation.go, chat_message.go
3. Create service: chat.go (or merge into existing ai service)
4. Create handler: chat.go
5. Register routes
6. Update frontend to call API instead of localStorage