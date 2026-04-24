# AI Chat UX Streaming Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add SSE-based streaming responses, conversation renaming, preserved conversation ordering, and near-bottom auto-scroll behavior to the AI chat experience.

**Architecture:** Keep the existing MySQL-backed conversation model and non-stream chat API intact, but add a dedicated SSE endpoint for interactive chat. The backend will persist the user message immediately, stream assistant chunks to the frontend, then persist a single final assistant message after completion. The frontend store will switch from one-shot assistant insertion to placeholder-message chunk appending, while UI-only rename and scroll-follow state stays in the page component.

**Tech Stack:** Go, Gin, Redis-backed RAG flow, Eino/Ark chat model, React, Zustand, fetch + ReadableStream SSE parsing, TypeScript

---

## File Structure

**Backend files**
- Modify: `/Users/lizhuang/go/src/wenDao/backend/internal/pkg/eino/llm.go` — add stream-capable LLM abstraction and Ark implementation
- Modify: `/Users/lizhuang/go/src/wenDao/backend/internal/pkg/eino/chain.go` — add `ExecuteStream` and share routing logic with existing RAG/fallback path
- Modify: `/Users/lizhuang/go/src/wenDao/backend/internal/service/ai.go` — expose stream orchestration and final assistant persistence behavior
- Modify: `/Users/lizhuang/go/src/wenDao/backend/internal/handler/ai.go` — add `ChatStream` SSE handler
- Modify: `/Users/lizhuang/go/src/wenDao/backend/internal/handler/chat.go` — add rename endpoint
- Modify: `/Users/lizhuang/go/src/wenDao/backend/internal/repository/conversation.go` — add focused title update method if needed
- Modify: `/Users/lizhuang/go/src/wenDao/backend/cmd/server/main.go` — register stream route and rename route

**Frontend files**
- Modify: `/Users/lizhuang/go/src/wenDao/frontend/src/api/chat.ts` — add SSE stream helper and rename API
- Modify: `/Users/lizhuang/go/src/wenDao/frontend/src/store/chatStore.ts` — stream-aware send flow and post-stream state updates
- Modify: `/Users/lizhuang/go/src/wenDao/frontend/src/pages/AIChat.tsx` — rename UI, scroll tracking, stream-follow behavior
- Modify: `/Users/lizhuang/go/src/wenDao/frontend/src/i18n.ts` — add rename-related copy

---

### Task 1: Add backend stream-capable LLM interface

**Files:**
- Modify: `/Users/lizhuang/go/src/wenDao/backend/internal/pkg/eino/llm.go`

- [ ] **Step 1: Add a streaming method to the LLM interface**

Update the interface in `backend/internal/pkg/eino/llm.go`.

```go
type LLMClient interface {
	Chat(messages []ChatMessage) (string, error)
	Stream(messages []ChatMessage, onChunk func(string) error) error
	GetModel() model.ChatModel
}
```

- [ ] **Step 2: Add a stream implementation to `arkLLMClient`**

In the same file, add a method that converts `[]ChatMessage` to `[]*schema.Message`, opens a stream on the underlying Ark/Eino model, and forwards each non-empty chunk to `onChunk`.

Use this structure:

```go
func (c *arkLLMClient) Stream(messages []ChatMessage, onChunk func(string) error) error {
	ctx := context.Background()
	schemaMessages := make([]*schema.Message, 0, len(messages))
	for _, msg := range messages {
		schemaMessages = append(schemaMessages, &schema.Message{
			Role:    schema.RoleType(msg.Role),
			Content: msg.Content,
		})
	}

	reader, err := c.client.Stream(ctx, schemaMessages, model.WithTemperature(c.temperature), model.WithMaxTokens(c.maxTokens))
	if err != nil {
		return fmt.Errorf("failed to stream response: %w", err)
	}
	defer reader.Close()

	for {
		msg, err := reader.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to receive stream chunk: %w", err)
		}
		if msg != nil && msg.Content != "" {
			if err := onChunk(msg.Content); err != nil {
				return err
			}
		}
	}

	return nil
}
```

- [ ] **Step 3: Add required imports**

Ensure `backend/internal/pkg/eino/llm.go` imports `io` after adding the streaming method.

```go
import (
	"context"
	"fmt"
	"io"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"wenDao/config"
)
```

- [ ] **Step 4: Run backend build to verify interface compiles**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/backend && go build ./...
```

Expected: the build fails only if downstream call sites still need to be updated. If so, continue immediately to Task 2 before re-running.

- [ ] **Step 5: Commit**

```bash
git add /Users/lizhuang/go/src/wenDao/backend/internal/pkg/eino/llm.go
git commit -m "feat: add stream-capable llm interface"
```

---

### Task 2: Add streaming execution path to the RAG chain

**Files:**
- Modify: `/Users/lizhuang/go/src/wenDao/backend/internal/pkg/eino/chain.go`

- [ ] **Step 1: Keep `Execute` intact and add `buildMessages` helper**

Refactor prompt selection into a shared helper so the sync and streaming paths do not duplicate routing logic.

```go
func (c *RAGChain) buildMessages(question string) ([]*schema.Message, error) {
	docs, err := c.retriever.Retrieve(context.Background(), question)
	if err != nil {
		return nil, fmt.Errorf("retrieval failed: %w", err)
	}

	if shouldUseFallback(docs, c.ragMinScore) {
		return c.buildFallbackMessages(question), nil
	}

	return c.buildRAGMessages(question, docs), nil
}
```

- [ ] **Step 2: Update `Execute` to reuse the shared helper**

Replace the retrieval/routing block in `Execute` with:

```go
func (c *RAGChain) Execute(ctx context.Context, question string) (string, error) {
	messages, err := c.buildMessages(question)
	if err != nil {
		return "", err
	}

	resp, err := c.llm.Generate(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("generation failed: %w", err)
	}

	return resp.Content, nil
}
```

- [ ] **Step 3: Add `ExecuteStream`**

Add a streaming variant that reuses the same prompt selection.

```go
func (c *RAGChain) ExecuteStream(ctx context.Context, question string, onChunk func(string) error) error {
	messages, err := c.buildMessages(question)
	if err != nil {
		return err
	}

	reader, err := c.llm.Stream(ctx, messages)
	if err != nil {
		return fmt.Errorf("generation failed: %w", err)
	}
	defer reader.Close()

	for {
		chunk, err := reader.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("stream receive failed: %w", err)
		}
		if chunk != nil && chunk.Content != "" {
			if err := onChunk(chunk.Content); err != nil {
				return err
			}
		}
	}
}
```

If the final implementation uses `LLMClient.Stream(messages, onChunk)` instead of direct model streaming, keep the same shape but call that method instead:

```go
func (c *RAGChain) ExecuteStream(ctx context.Context, question string, onChunk func(string) error) error {
	messages, err := c.buildMessages(question)
	if err != nil {
		return err
	}
	return c.llmClient.Stream(messages, onChunk)
}
```

Choose one concrete version and keep it consistent with Task 1.

- [ ] **Step 4: Add imports needed by the streaming path**

If the chosen implementation needs `io`, add it explicitly.

- [ ] **Step 5: Run backend build**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/backend && go build ./...
```

Expected: build may still fail until Task 3 updates service call sites. Continue immediately if so.

- [ ] **Step 6: Commit**

```bash
git add /Users/lizhuang/go/src/wenDao/backend/internal/pkg/eino/chain.go
 git commit -m "feat: add streaming execution to rag chain"
```

---

### Task 3: Add AI service streaming orchestration and persistence

**Files:**
- Modify: `/Users/lizhuang/go/src/wenDao/backend/internal/service/ai.go`

- [ ] **Step 1: Extend the service interface**

Add a streaming method in `backend/internal/service/ai.go`.

```go
type AIService interface {
	Chat(question string, conversationID *int64, userID *int64) (string, error)
	ChatStream(question string, conversationID *int64, userID *int64, onChunk func(string) error) (string, error)
}
```

- [ ] **Step 2: Add a shared conversation validation helper**

Introduce a focused helper to avoid repeating ownership checks.

```go
func (s *aiService) validateConversation(conversationID *int64, userID *int64) (*model.Conversation, error) {
	if conversationID == nil || *conversationID <= 0 {
		return nil, nil
	}
	if userID == nil {
		return nil, fmt.Errorf("user authentication required")
	}

	conv, err := s.convRepo.GetByID(*conversationID)
	if err != nil {
		return nil, fmt.Errorf("conversation not found: %w", err)
	}
	if conv.UserID != *userID {
		return nil, fmt.Errorf("conversation access denied")
	}
	return conv, nil
}
```

- [ ] **Step 3: Add a helper for updating conversation metadata**

```go
func (s *aiService) updateConversationMetadata(conv *model.Conversation, question string) {
	if conv == nil {
		return
	}
	conv.UpdatedAt = time.Now()
	if conv.Title == "" || conv.Title == "New Conversation" || conv.Title == "新会话" {
		conv.Title = buildConversationTitle(question)
	}
	if err := s.convRepo.Update(conv); err != nil {
		s.logger.Warn("Failed to update conversation metadata", zap.Error(err))
	}
}
```

- [ ] **Step 4: Implement `ChatStream`**

Add a stream-aware method that:
- validates conversation ownership,
- persists the user message immediately,
- buffers assistant chunks in memory,
- persists one final assistant message after stream completion,
- returns the full assistant answer.

```go
func (s *aiService) ChatStream(question string, conversationID *int64, userID *int64, onChunk func(string) error) (string, error) {
	conv, err := s.validateConversation(conversationID, userID)
	if err != nil {
		return "", err
	}

	if conversationID != nil && *conversationID > 0 {
		userMsg := &model.ChatMessage{
			ConversationID: *conversationID,
			Role:           "user",
			Content:        question,
		}
		if err := s.msgRepo.Create(userMsg); err != nil {
			s.logger.Warn("Failed to save user message", zap.Error(err))
		}
	}

	var builder strings.Builder
	err = s.ragChain.ExecuteStream(context.Background(), question, func(chunk string) error {
		builder.WriteString(chunk)
		return onChunk(chunk)
	})
	if err != nil {
		s.logger.Error("RAG stream execution failed", zap.Error(err))
		return "", fmt.Errorf("AI service unavailable: %w", err)
	}

	answer := builder.String()
	if conversationID != nil && *conversationID > 0 {
		assistantMsg := &model.ChatMessage{
			ConversationID: *conversationID,
			Role:           "assistant",
			Content:        answer,
		}
		if err := s.msgRepo.Create(assistantMsg); err != nil {
			s.logger.Warn("Failed to save assistant message", zap.Error(err))
		}
		s.updateConversationMetadata(conv, question)
	}

	return answer, nil
}
```

- [ ] **Step 5: Refactor existing `Chat` to reuse validation helpers**

Replace inline ownership logic with `validateConversation` and `updateConversationMetadata` so both paths stay consistent.

- [ ] **Step 6: Run backend build**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/backend && go build ./...
```

Expected: build may still fail until handler wiring is added in Task 4. Continue immediately if so.

- [ ] **Step 7: Commit**

```bash
git add /Users/lizhuang/go/src/wenDao/backend/internal/service/ai.go
git commit -m "feat: add ai streaming service flow"
```

---

### Task 4: Add SSE endpoint and conversation rename endpoint

**Files:**
- Modify: `/Users/lizhuang/go/src/wenDao/backend/internal/handler/ai.go`
- Modify: `/Users/lizhuang/go/src/wenDao/backend/internal/handler/chat.go`
- Modify: `/Users/lizhuang/go/src/wenDao/backend/cmd/server/main.go`

- [ ] **Step 1: Add conversation rename request DTO and handler**

In `backend/internal/handler/chat.go`, add:

```go
type UpdateConversationRequest struct {
	Title string `json:"title" binding:"required,min=1,max=255"`
}
```

Add a new handler method:

```go
func (h *ChatHandler) Update(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "Missing user ID")
		return
	}

	convID := c.Param("id")
	var convIDInt int64
	if _, err := fmt.Sscanf(convID, "%d", &convIDInt); err != nil {
		response.InvalidParams(c, "Invalid conversation ID")
		return
	}

	conv, err := h.convRepo.GetByID(convIDInt)
	if err != nil {
		response.NotFound(c, "Conversation not found")
		return
	}
	if conv.UserID != userID.(int64) {
		response.Forbidden(c, "Access denied")
		return
	}

	var req UpdateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "Invalid title")
		return
	}

	conv.Title = req.Title
	if err := h.convRepo.Update(conv); err != nil {
		response.InternalError(c, "Failed to update conversation")
		return
	}

	response.Success(c, ConversationResponse{
		ID:        conv.ID,
		UserID:    conv.UserID,
		Title:     conv.Title,
		CreatedAt: conv.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt: conv.UpdatedAt.Format("2006-01-02 15:04:05"),
	})
}
```

- [ ] **Step 2: Add SSE stream handler**

In `backend/internal/handler/ai.go`, add a streaming handler that sets SSE headers and emits `start`, `chunk`, `done`, and `error` events.

```go
func (h *AIHandler) ChatStream(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParams(c, "消息内容不能为空")
		return
	}

	var userID *int64
	if uid, exists := c.Get("user_id"); exists {
		if v, ok := uid.(int64); ok {
			userID = &v
		}
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		response.InternalError(c, "Streaming unsupported")
		return
	}

	writeEvent := func(event string, data string) error {
		_, err := fmt.Fprintf(c.Writer, "event: %s\n", event)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		if err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	if err := writeEvent("start", `{"conversation_id":`+strconv.FormatInt(*req.ConversationID, 10)+`}`); err != nil {
		return
	}

	answer, err := h.aiService.ChatStream(req.Message, req.ConversationID, userID, func(chunk string) error {
		payload, _ := json.Marshal(gin.H{"content": chunk})
		return writeEvent("chunk", string(payload))
	})
	if err != nil {
		payload, _ := json.Marshal(gin.H{"message": "生成回答失败，请稍后再试"})
		_ = writeEvent("error", string(payload))
		return
	}

	payload, _ := json.Marshal(gin.H{"message": answer})
	_ = writeEvent("done", string(payload))
}
```

Keep the event names exactly as specified in the design.

- [ ] **Step 3: Add required imports in `ai.go`**

```go
import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"wenDao/internal/pkg/response"
	"wenDao/internal/service"
)
```

- [ ] **Step 4: Register new routes in `main.go`**

Add the stream route under `/api/ai` and the rename route under `/api/chat/conversations`.

```go
ai := api.Group("/ai")
ai.Use(middleware.AuthRequired(cfg.JWT.Secret, rdb))
{
	ai.POST("/chat", aiHandler.Chat)
	ai.POST("/chat/stream", aiHandler.ChatStream)
}

conversations := api.Group("/chat/conversations")
conversations.Use(middleware.AuthRequired(cfg.JWT.Secret, rdb))
{
	conversations.GET("", chatHandler.List)
	conversations.POST("", chatHandler.Create)
	conversations.GET(":id", chatHandler.Get)
	conversations.PATCH(":id", chatHandler.Update)
	conversations.DELETE(":id", chatHandler.Delete)
}
```

Use the exact slash style already present in the file.

- [ ] **Step 5: Run backend build**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/backend && go build ./...
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add /Users/lizhuang/go/src/wenDao/backend/internal/handler/ai.go /Users/lizhuang/go/src/wenDao/backend/internal/handler/chat.go /Users/lizhuang/go/src/wenDao/backend/cmd/server/main.go
git commit -m "feat: add ai streaming endpoint and conversation rename api"
```

---

### Task 5: Add frontend streaming API and rename API

**Files:**
- Modify: `/Users/lizhuang/go/src/wenDao/frontend/src/api/chat.ts`

- [ ] **Step 1: Add rename API call**

In `frontend/src/api/chat.ts`, add:

```ts
renameConversation: (id: number, title: string) => {
  return request.patch<any>('/chat/conversations/' + id, { title });
},
```

- [ ] **Step 2: Add POST-based SSE stream helper**

Add a helper that uses `fetch` and parses `text/event-stream` from `response.body`.

```ts
type StreamHandlers = {
  onStart?: (payload: any) => void;
  onChunk: (payload: { content: string }) => void;
  onDone?: (payload: any) => void;
  onError?: (payload: any) => void;
};

async function readSSEStream(response: Response, handlers: StreamHandlers) {
  const reader = response.body?.getReader();
  if (!reader) throw new Error('No response body');

  const decoder = new TextDecoder('utf-8');
  let buffer = '';

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    buffer += decoder.decode(value, { stream: true });
    const events = buffer.split('\n\n');
    buffer = events.pop() || '';

    for (const raw of events) {
      const lines = raw.split('\n');
      const eventLine = lines.find((l) => l.startsWith('event: '));
      const dataLine = lines.find((l) => l.startsWith('data: '));
      if (!eventLine || !dataLine) continue;

      const eventName = eventLine.replace('event: ', '').trim();
      const payload = JSON.parse(dataLine.replace('data: ', '').trim());

      if (eventName === 'start') handlers.onStart?.(payload);
      if (eventName === 'chunk') handlers.onChunk(payload);
      if (eventName === 'done') handlers.onDone?.(payload);
      if (eventName === 'error') handlers.onError?.(payload);
    }
  }
}
```

- [ ] **Step 3: Add `streamMessage` wrapper**

Use the frontend token already stored in localStorage.

```ts
streamMessage: async (data: ChatRequest, handlers: StreamHandlers) => {
  const token = localStorage.getItem('access_token');
  const response = await fetch('/api/ai/chat/stream', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify(data),
  });

  if (!response.ok) {
    throw new Error('流式请求失败');
  }

  await readSSEStream(response, handlers);
},
```

If your project already uses `VITE_API_BASE_URL`, build the URL from that base instead of hardcoding `/api`.

- [ ] **Step 4: Verify TypeScript compilation for the API file via full frontend build**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/frontend && npm run build
```

Expected: it may still fail until Tasks 6 and 7 are completed. Continue immediately if so.

- [ ] **Step 5: Commit**

```bash
git add /Users/lizhuang/go/src/wenDao/frontend/src/api/chat.ts
git commit -m "feat: add chat streaming and rename api helpers"
```

---

### Task 6: Make the chat store stream-aware

**Files:**
- Modify: `/Users/lizhuang/go/src/wenDao/frontend/src/store/chatStore.ts`

- [ ] **Step 1: Extend state with streaming flags**

Add the new fields to `ChatState` and initial state.

```ts
interface ChatState {
  conversations: Record<number, Conversation>;
  activeId: number | null;
  isTyping: boolean;
  isStreaming: boolean;
  streamingConversationId: number | null;

  loadConversations: () => Promise<void>;
  createNewChat: () => Promise<void>;
  setActiveChat: (id: number) => Promise<void>;
  deleteChat: (id: number) => Promise<void>;
  renameChat: (id: number, title: string) => Promise<void>;
  sendMessage: (content: string) => Promise<void>;
  clearMessages: () => void;
}
```

Initial state:

```ts
isStreaming: false,
streamingConversationId: null,
```

- [ ] **Step 2: Add `renameChat` action**

```ts
renameChat: async (id, title) => {
  try {
    const response = await chatApi.renameConversation(id, title);
    set((state) => ({
      conversations: {
        ...state.conversations,
        [id]: {
          ...state.conversations[id],
          title: response.title,
          updatedAt: new Date(response.updated_at).getTime(),
        },
      },
    }));
  } catch (error) {
    console.error('Failed to rename conversation:', error);
    throw error;
  }
},
```

- [ ] **Step 3: Replace one-shot assistant insertion with placeholder + stream updates**

Inside `sendMessage`, replace the current `chatApi.sendMessage(...)` block with:

```ts
const assistantMessageId = (Date.now() + 1).toString();
const assistantPlaceholder: ChatMessage = {
  id: assistantMessageId,
  role: 'assistant',
  content: '',
  timestamp: Date.now(),
};

set((state) => ({
  conversations: {
    ...state.conversations,
    [currentId!]: {
      ...state.conversations[currentId!],
      title: nextTitle,
      messages: [...state.conversations[currentId!].messages, assistantPlaceholder],
      updatedAt: Date.now(),
    },
  },
  isTyping: true,
  isStreaming: true,
  streamingConversationId: currentId!,
}));

try {
  await chatApi.streamMessage(
    { message: content, conversation_id: currentId },
    {
      onChunk: ({ content: chunk }) => {
        set((state) => ({
          conversations: {
            ...state.conversations,
            [currentId!]: {
              ...state.conversations[currentId!],
              messages: state.conversations[currentId!].messages.map((msg) =>
                msg.id === assistantMessageId
                  ? { ...msg, content: msg.content + chunk }
                  : msg
              ),
              updatedAt: Date.now(),
            },
          },
        }));
      },
      onDone: () => {
        set({
          isTyping: false,
          isStreaming: false,
          streamingConversationId: null,
        });
      },
      onError: (payload) => {
        set((state) => ({
          conversations: {
            ...state.conversations,
            [currentId!]: {
              ...state.conversations[currentId!],
              messages: state.conversations[currentId!].messages.map((msg) =>
                msg.id === assistantMessageId
                  ? {
                      ...msg,
                      content: msg.content || payload.message || '生成回答失败，请稍后再试。',
                    }
                  : msg
              ),
            },
          },
          isTyping: false,
          isStreaming: false,
          streamingConversationId: null,
        }));
      },
    }
  );
} catch (error) {
  set((state) => ({
    conversations: {
      ...state.conversations,
      [currentId!]: {
        ...state.conversations[currentId!],
        messages: state.conversations[currentId!].messages.map((msg) =>
          msg.id === assistantMessageId
            ? { ...msg, content: '生成回答失败，请稍后再试。' }
            : msg
        ),
      },
    },
    isTyping: false,
    isStreaming: false,
    streamingConversationId: null,
  }));
}
```

- [ ] **Step 4: Keep the current sort logic unchanged**

Do not alter the existing comparator in the store or page. The only required behavior is that `updatedAt` keeps changing during send and stream completion so the current sorting still works.

- [ ] **Step 5: Run frontend build**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/frontend && npm run build
```

Expected: may still fail until AIChat UI changes are completed in Task 7. Continue immediately if so.

- [ ] **Step 6: Commit**

```bash
git add /Users/lizhuang/go/src/wenDao/frontend/src/store/chatStore.ts
git commit -m "feat: make chat store stream assistant replies"
```

---

### Task 7: Add rename UI and near-bottom auto-scroll behavior

**Files:**
- Modify: `/Users/lizhuang/go/src/wenDao/frontend/src/pages/AIChat.tsx`
- Modify: `/Users/lizhuang/go/src/wenDao/frontend/src/i18n.ts`

- [ ] **Step 1: Add rename translations**

In both language blocks in `frontend/src/i18n.ts`, add:

```ts
chat: {
  // existing keys...
  rename: 'Rename',
  saveName: 'Save Name',
  cancelRename: 'Cancel',
  renamePlaceholder: 'Conversation title',
}
```

Chinese block:

```ts
chat: {
  // existing keys...
  rename: '重命名',
  saveName: '保存名称',
  cancelRename: '取消',
  renamePlaceholder: '会话名称',
}
```

- [ ] **Step 2: Add local rename state and scroll refs in `AIChat.tsx`**

At the top of the component, add:

```ts
const [isRenaming, setIsRenaming] = useState(false);
const [draftTitle, setDraftTitle] = useState('');
const scrollContainerRef = useRef<HTMLDivElement>(null);
const [isNearBottom, setIsNearBottom] = useState(true);
const { conversations, activeId, isTyping, isStreaming, loadConversations, sendMessage, createNewChat, setActiveChat, deleteChat, renameChat } = useChatStore();
```

Add an effect to sync `draftTitle` when active conversation changes:

```ts
useEffect(() => {
  if (activeChat) {
    setDraftTitle(activeChat.title);
    setIsRenaming(false);
  }
}, [activeChat?.id, activeChat?.title]);
```

- [ ] **Step 3: Track whether the user is near the bottom**

Attach the scroll container ref to the messages area and add:

```ts
const handleScroll = () => {
  const container = scrollContainerRef.current;
  if (!container) return;

  const distanceFromBottom = container.scrollHeight - container.scrollTop - container.clientHeight;
  setIsNearBottom(distanceFromBottom <= 80);
};
```

Then update the messages area container:

```tsx
<div
  ref={scrollContainerRef}
  onScroll={handleScroll}
  className="flex-1 overflow-y-auto px-10 py-10 space-y-8 scrollbar-hide relative bg-neutral-50/30 dark:bg-neutral-800/50"
>
```

- [ ] **Step 4: Auto-scroll only when the user is near bottom**

Add this effect:

```ts
useEffect(() => {
  const container = scrollContainerRef.current;
  if (!container || !isNearBottom) return;
  container.scrollTop = container.scrollHeight;
}, [messages, isTyping, isStreaming, isNearBottom]);
```

- [ ] **Step 5: Add inline rename UI in the header**

Replace the current title block with editable UI.

```tsx
<div>
  {isRenaming && activeChat ? (
    <div className="flex items-center gap-3">
      <input
        value={draftTitle}
        onChange={(e) => setDraftTitle(e.target.value)}
        placeholder={t('chat.renamePlaceholder')}
        className="bg-transparent border border-neutral-200 dark:border-neutral-600 rounded-lg px-3 py-2 text-lg font-serif font-black text-neutral-900 dark:text-neutral-100"
        onKeyDown={async (e) => {
          if (e.key === 'Enter' && draftTitle.trim()) {
            await renameChat(activeChat.id, draftTitle.trim());
            setIsRenaming(false);
          }
          if (e.key === 'Escape') {
            setDraftTitle(activeChat.title);
            setIsRenaming(false);
          }
        }}
        autoFocus
      />
      <button
        onClick={async () => {
          if (!draftTitle.trim()) return;
          await renameChat(activeChat.id, draftTitle.trim());
          setIsRenaming(false);
        }}
        className="text-xs text-primary-600 dark:text-primary-400"
      >
        {t('chat.saveName')}
      </button>
      <button
        onClick={() => {
          setDraftTitle(activeChat.title);
          setIsRenaming(false);
        }}
        className="text-xs text-neutral-500 dark:text-neutral-400"
      >
        {t('chat.cancelRename')}
      </button>
    </div>
  ) : (
    <div className="flex items-center gap-3">
      <h2 className="text-lg font-serif font-black text-neutral-900 dark:text-neutral-100">
        {activeChat?.title || t('chat.title')}
      </h2>
      {activeChat && (
        <button
          onClick={() => setIsRenaming(true)}
          className="text-xs text-neutral-400 hover:text-primary-600 dark:hover:text-primary-400"
        >
          {t('chat.rename')}
        </button>
      )}
    </div>
  )}
</div>
```

- [ ] **Step 6: Keep the existing sort rule exactly as-is**

Do not change the current comparator in `AIChat.tsx`.

This existing code must remain:

```ts
.sort((a, b) => {
  if (a.messages.length === 0 && b.messages.length > 0) return -1;
  if (b.messages.length === 0 && a.messages.length > 0) return 1;
  return b.updatedAt - a.updatedAt;
})
```

- [ ] **Step 7: Run frontend build**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/frontend && npm run build
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add /Users/lizhuang/go/src/wenDao/frontend/src/pages/AIChat.tsx /Users/lizhuang/go/src/wenDao/frontend/src/i18n.ts
git commit -m "feat: add streaming chat ui rename and autoscroll"
```

---

### Task 8: End-to-end manual verification

**Files:**
- Modify: none unless fixes are discovered

- [ ] **Step 1: Start backend**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/backend && go run ./cmd/server
```

Expected: server starts successfully and exposes `/api/ai/chat/stream`.

- [ ] **Step 2: Start frontend**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/frontend && npm run dev
```

Expected: frontend starts and `/ai-chat` loads.

- [ ] **Step 3: Verify streaming visually**

Open `/ai-chat`, send a question, and confirm:
- the assistant bubble appears immediately,
- text grows progressively,
- response is not inserted all at once.

- [ ] **Step 4: Verify rename behavior**

Open an existing conversation, click rename in the header, change the title, press Enter, then refresh the page.

Expected:
- title updates immediately,
- title persists after refresh,
- another user could not rename it through the API.

- [ ] **Step 5: Verify auto-scroll behavior**

Test two cases:
1. stay near the bottom while a long answer streams — the viewport should follow automatically
2. scroll upward intentionally while the answer streams — the viewport should stop forcing scroll

- [ ] **Step 6: Verify sorting behavior**

Create a new empty conversation and confirm:
- it appears at the top of the list,
- after sending the first message it becomes non-empty,
- ordering among non-empty conversations is newest first.

- [ ] **Step 7: Fix any issues found and re-run builds**

Run:
```bash
cd /Users/lizhuang/go/src/wenDao/backend && go build ./...
cd /Users/lizhuang/go/src/wenDao/frontend && npm run build
```

Expected: PASS.

- [ ] **Step 8: Commit only if verification required fixes**

If changes were needed:

```bash
git add <exact changed files>
git commit -m "fix: polish ai chat streaming ux verification issues"
```

---

## Self-Review

### Spec coverage
- SSE stream endpoint: Task 4
- stream-capable LLM: Task 1
- stream-aware chain: Task 2
- service-side persistence flow: Task 3
- rename endpoint: Task 4
- frontend stream handling: Task 5 and Task 6
- preserved sorting rule: Task 6
- auto-scroll near bottom only: Task 7
- validation and manual verification: Task 8

No gaps found.

### Placeholder scan
- No TODO/TBD placeholders remain.
- All file paths are explicit.
- All commands are concrete.
- Method and event names are named consistently: `ChatStream`, `streamMessage`, `renameConversation`, `start/chunk/done/error`.

### Type consistency
- Backend stream method names are consistent across handler/service/plan.
- Frontend rename method names are consistent across api/store/ui.
- The sort comparator is intentionally preserved verbatim.
- The event protocol is stable between backend and frontend.
