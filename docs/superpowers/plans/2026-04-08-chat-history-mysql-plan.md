# Chat History MySQL Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Store AI chat conversation history in MySQL instead of browser localStorage

**Architecture:** Create conversations and chat_messages tables in MySQL, add backend API endpoints, update frontend to call API

**Tech Stack:** Go + Gin + GORM, MySQL

---

### Task 1: Create Model Files

**Files:**
- Create: `backend/internal/model/conversation.go`
- Create: `backend/internal/model/chat_message.go`

- [ ] **Step 1: Create conversation.go model**

```go
package model

import "time"

// Conversation 对话
type Conversation struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID   int64     `gorm:"not null;index" json:"user_id"`
	Title    string    `gorm:"size:255" json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Conversation) TableName() string {
	return "conversations"
}
```

- [ ] **Step 2: Create chat_message.go model**

```go
package model

import "time"

// ChatMessage 对话消息
type ChatMessage struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ConversationID int64    `gorm:"not null;index" json:"conversation_id"`
	Role           string   `gorm:"size:20;not null" json:"role"` // user, assistant
	Content        string   `gorm:"type:text" json:"content"`
	CreatedAt      time.Time `json:"created_at"`
}

func (ChatMessage) TableName() string {
	return "chat_messages"
}
```

- [ ] **Step 3: Commit**
```bash
git add backend/internal/model/conversation.go backend/internal/model/chat_message.go
git commit -m "feat: add conversation and chat_message models"
```

---

### Task 2: Create Repository Files

**Files:**
- Create: `backend/internal/repository/conversation.go`
- Create: `backend/internal/repository/chat_message.go`

- [ ] **Step 1: Create conversation repository**

```go
package repository

import (
	"wenDao/internal/model"

	"gorm.io/gorm"
)

type ConversationRepository struct {
	db *gorm.DB
}

func NewConversationRepository(db *gorm.DB) *ConversationRepository {
	return &ConversationRepository{db: db}
}

func (r *ConversationRepository) Create(conv *model.Conversation) error {
	return r.db.Create(conv).Error
}

func (r *ConversationRepository) GetByID(id int64) (*model.Conversation, error) {
	var conv model.Conversation
	err := r.db.First(&conv, id).Error
	return &conv, err
}

func (r *ConversationRepository) GetByUserID(userID int64) ([]model.Conversation, error) {
	var conversations []model.Conversation
	err := r.db.Where("user_id = ?", userID).Order("updated_at DESC").Find(&conversations).Error
	return conversations, err
}

func (r *ConversationRepository) Update(conv *model.Conversation) error {
	return r.db.Save(conv).Error
}

func (r *ConversationRepository) Delete(id int64) error {
	return r.db.Delete(&model.Conversation{}, id).Error
}
```

- [ ] **Step 2: Create chat_message repository**

```go
package repository

import (
	"wenDao/internal/model"

	"gorm.io/gorm"
)

type ChatMessageRepository struct {
	db *gorm.DB
}

func NewChatMessageRepository(db *gorm.DB) *ChatMessageRepository {
	return &ChatMessageRepository{db: db}
}

func (r *ChatMessageRepository) Create(msg *model.ChatMessage) error {
	return r.db.Create(msg).Error
}

func (r *ChatMessageRepository) GetByConversationID(conversationID int64) ([]model.ChatMessage, error) {
	var messages []model.ChatMessage
	err := r.db.Where("conversation_id = ?", conversationID).Order("created_at ASC").Find(&messages).Error
	return messages, err
}

func (r *ChatMessageRepository) DeleteByConversationID(conversationID int64) error {
	return r.db.Where("conversation_id = ?", conversationID).Delete(&model.ChatMessage{}).Error
}
```

- [ ] **Step 3: Commit**
```bash
git add backend/internal/repository/conversation.go backend/internal/repository/chat_message.go
git commit -m "feat: add conversation and chat_message repositories"
```

---

### Task 3: Create Handler and Service

**Files:**
- Create: `backend/internal/handler/chat.go`
- Modify: `backend/cmd/server/main.go` (register routes)

- [ ] **Step 1: Create chat handler**

```go
package handler

import (
	"net/http"
	"wenDao/internal/model"
	"wenDao/internal/repository"
	"wenDao/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type ChatHandler struct {
	convRepo    *repository.ConversationRepository
	msgRepo    *repository.ChatMessageRepository
}

func NewChatHandler(convRepo *repository.ConversationRepository, msgRepo *repository.ChatMessageRepository) *ChatHandler {
	return &ChatHandler{
		convRepo: convRepo,
		msgRepo:  msgRepo,
	}
}

// List 获取用户的对话列表
func (h *ChatHandler) List(c *gin.Context) {
	userID := c.GetInt64("user_id")

	convs, err := h.convRepo.GetByUserID(userID)
	if err != nil {
		response.InternalError(c, "Failed to get conversations")
		return
	}

	response.Success(c, convs)
}

// Create 创建新对话
func (h *ChatHandler) Create(c *gin.Context) {
	userID := c.GetInt64("user_id")

	var req struct {
		Title string `json:"title"`
	}
	c.ShouldBindJSON(&req)

	if req.Title == "" {
		req.Title = "New Conversation"
	}

	conv := &model.Conversation{
		UserID: userID,
		Title:  req.Title,
	}

	if err := h.convRepo.Create(conv); err != nil {
		response.InternalError(c, "Failed to create conversation")
		return
	}

	response.Success(c, conv)
}

// Get 获取对话详情（含消息）
func (h *ChatHandler) Get(c *gin.Context) {
	id := c.Param("id")
	var convID int64
	_, _ = scan, err := fmt.Sscanf(id, "%d", &convID)
	if err != nil {
		response.InvalidParams(c, "Invalid conversation ID")
		return
	}

	conv, err := h.convRepo.GetByID(convID)
	if err != nil {
		response.NotFound(c, "Conversation not found")
		return
	}

	// 检查权限
	userID := c.GetInt64("user_id")
	if conv.UserID != userID {
		response.Forbidden(c, "No permission")
		return
	}

	// 获取消息
	messages, _ := h.msgRepo.GetByConversationID(convID)

	response.Success(c, gin.H{
		"id":      conv.ID,
		"title":   conv.Title,
		"messages": messages,
	})
}

// Delete 删除对话
func (h *ChatHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	var convID int64
	_, _ = scan, err := fmt.Sscanf(id, "%d", &convID)
	if err != nil {
		response.InvalidParams(c, "Invalid conversation ID")
		return
	}

	// 检查存在和权限
	conv, err := h.convRepo.GetByID(convID)
	if err != nil {
		response.NotFound(c, "Conversation not found")
		return
	}

	userID := c.GetInt64("user_id")
	if conv.UserID != userID {
		response.Forbidden(c, "No permission")
		return
	}

	// 删除消息
	h.msgRepo.DeleteByConversationID(convID)
	// 删除对话
	h.convRepo.Delete(convID)

	response.Success(c, nil)
}
```

Note: This handler doesn't include the AI chat functionality - that's handled separately by the existing AI service. The frontend will send messages directly to the existing AI endpoint.

- [ ] **Step 2: Add fmt import to chat handler**
```go
import (
	"fmt"  // add this
	// ...
)
```

- [ ] **Step 3: Register routes in main.go**

Find where routes are registered and add:
```go
chatHandler := handler.NewChatHandler(convRepo, msgRepo)
chats := api.Group("/chat")
chats.GET("/conversations", chatHandler.List)
chats.POST("/conversations", chatHandler.Create)
chats.GET("/conversations/:id", chatHandler.Get)
chats.DELETE("/conversations/:id", chatHandler.Delete)
```

- [ ] **Step 4: Commit**
```bash
git add backend/internal/handler/chat.go
git commit -m "feat: add chat handler with CRUD API"
```

---

### Task 4: Create MySQL Tables

- [ ] **Step 1: Run migrations**

```bash
# Manually create tables via SQL or use GORM auto-migrate
```

```sql
CREATE TABLE conversations (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    title VARCHAR(255),
    created_at DATETIME,
    updated_at DATETIME,
    INDEX idx_user_id (user_id)
);

CREATE TABLE chat_messages (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    conversation_id BIGINT NOT NULL,
    role VARCHAR(20) NOT NULL,
    content TEXT,
    created_at DATETIME,
    INDEX idx_conversation_id (conversation_id)
);
```

Or add auto-migrate in main.go:
```go
db.AutoMigrate(&model.Conversation{}, &model.ChatMessage{})
```

- [ ] **Step 2: Commit**
```bash
git commit -m "feat: add database migrations for chat tables"
```

---

### Task 5: Update Frontend API Client

**Files:**
- Modify: `frontend/src/api/chat.ts`
- Modify: `frontend/src/store/chatStore.ts`

- [ ] **Step 1: Add API methods in chat.ts**

```typescript
// Get conversation list
export const getConversations = () => client.get('/chat/conversations');

// Create new conversation
export const createConversation = (title: string) =>
  client.post('/chat/conversations', { title });

// Get conversation with messages
export const getConversation = (id: number) =>
  client.get(`/chat/conversations/${id}`);

// Delete conversation
export const deleteConversation = (id: number) =>
  client.delete(`/chat/conversations/${id}`);
```

- [ ] **Step 2: Update chatStore to use API**

Replace localStorage logic with API calls:

```typescript
import {
  getConversations,
  createConversation,
  getConversation,
  deleteConversation
} from '@/api/chat';

createNewChat: async () => {
  const res = await createConversation('New Conversation');
  const newChat = res.data;
  set((state) => ({
    conversations: { ...state.conversations, [newChat.id]: newChat },
    activeId: newChat.id,
  }));
},

setActiveChat: async (id: string) => {
  set({ activeId: id });
  const res = await getConversation(Number(id));
  const chat = res.data;
  // Update messages from API response
  set((state) => ({
    conversations: {
      ...state.conversations,
      [id]: { ...chat, messages: chat.messages }
    }
  }));
},

deleteChat: async (id: string) => {
  await deleteConversation(Number(id));
  set((state) => {
    const newConvs = { ...state.conversations };
    delete newConvs[id];
    return {
      conversations: newConvs,
      activeId: state.activeId === id ? null : state.activeId,
    };
  });
},
```

- [ ] **Step 3: Commit**
```bash
git add frontend/src/api/chat.ts frontend/src/store/chatStore.ts
git commit -m "feat: update frontend to use backend API for chat history"
```

---

### Task 6: Testing

- [ ] **Step 1: Test API endpoints**

```bash
# Login first to get token
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"123456"}'

# Then test chat endpoints
curl -X GET http://localhost:8080/api/chat/conversations \
  -H "Authorization: Bearer TOKEN"

curl -X POST http://localhost:8080/api/chat/conversations \
  -H "Authorization: Bearer TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"Test Chat"}'
```

- [ ] **Step 2: Test frontend**

Open browser, login, go to AI Chat page, create a new conversation, send a message, verify it appears in history after refresh.

---

**Plan complete.**