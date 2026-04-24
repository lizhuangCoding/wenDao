# AI Chat UX Streaming Design

## Overview
Improve the AI chat experience in four coordinated areas:
1. replace one-shot assistant replies with SSE-based streaming output,
2. allow users to rename conversations,
3. preserve the existing sorting rule where empty conversations stay first and non-empty conversations are sorted by most recent update,
4. make the chat viewport automatically follow the assistant response while the user is near the bottom.

This design keeps the current chat persistence model in MySQL, keeps the existing non-streaming endpoint available if needed, and introduces a dedicated streaming path for the interactive chat UI.

## Goal
Make the AI assistant feel responsive and modern without changing the overall conversation data model. Users should see the assistant answer appear progressively, be able to rename sessions, keep the existing conversation ordering behavior, and avoid manually scrolling during long answers.

## Current Problems
### 1. Responses are one-shot
`frontend/src/store/chatStore.ts` currently waits for `chatApi.sendMessage()` to return a full `response.message`, then inserts the entire assistant message at once. The backend `backend/internal/handler/ai.go` also only returns JSON, not a stream.

### 2. Conversation rename is unsupported
The frontend only displays titles and has no rename interaction. The backend chat handler exposes list/create/get/delete but not update-title behavior.

### 3. Sorting is already correct and must be preserved
The frontend currently sorts conversations so empty conversations come first and non-empty conversations are sorted by `updatedAt` descending. This behavior must remain unchanged after the streaming changes.

### 4. Auto-scroll does not actually follow answers
The AI chat page has a `messagesEndRef`, but there is no active effect that scrolls the container while assistant chunks arrive. Once the conversation grows, the user must manually scroll to see the latest answer.

## Design Summary
We will add a dedicated streaming chat endpoint using SSE (`text/event-stream`) and update the frontend store to consume streamed chunks in real time. The existing persistence model remains the same:
- save the user message immediately,
- stream assistant chunks to the frontend,
- persist the final assistant message once streaming completes.

Conversation rename will be added as a focused CRUD extension to the existing conversation handler. Sorting stays as-is, but the state update rules will ensure streamed conversations move correctly once they become non-empty.

## Files Affected
### Backend
- Modify: `backend/internal/handler/ai.go`
- Modify: `backend/internal/service/ai.go`
- Modify: `backend/internal/pkg/eino/llm.go`
- Modify: `backend/internal/pkg/eino/chain.go`
- Modify: `backend/internal/handler/chat.go`
- Modify: `backend/internal/repository/conversation.go`
- Modify: `backend/cmd/server/main.go`

### Frontend
- Modify: `frontend/src/api/chat.ts`
- Modify: `frontend/src/store/chatStore.ts`
- Modify: `frontend/src/pages/AIChat.tsx`
- Modify: `frontend/src/i18n.ts`

## Backend API Design
### 1. Keep existing JSON endpoint
Keep the existing `POST /api/ai/chat` endpoint for compatibility or non-stream use.

### 2. Add streaming endpoint
Add:

```http
POST /api/ai/chat/stream
Content-Type: application/json
Accept: text/event-stream
```

Request body stays consistent with the existing chat request:

```json
{
  "message": "用户问题",
  "conversation_id": 123,
  "article_id": null
}
```

### 3. Add conversation rename endpoint
Add:

```http
PATCH /api/chat/conversations/:id
```

Request:

```json
{
  "title": "新的会话名称"
}
```

Only the conversation owner may rename the conversation.

## SSE Event Protocol
Use explicit named events so the frontend state machine remains simple.

### `start`
Sent once after the stream is initialized.

Example:
```text
event: start
data: {"conversation_id":123}
```

### `chunk`
Sent repeatedly as the assistant produces text.

Example:
```text
event: chunk
data: {"content":"这是新增的一段文本"}
```

### `done`
Sent once when the answer is complete and the final assistant message has been persisted.

Example:
```text
event: done
data: {"message_id":456}
```

### `error`
Sent if generation fails after the stream has started.

Example:
```text
event: error
data: {"message":"生成回答失败，请稍后再试"}
```

## Streaming Backend Flow
The new streaming path should work like this:

1. Validate request and conversation ownership.
2. Save the user message to MySQL immediately.
3. Build the final prompt path:
   - RAG prompt if Redis retrieval score passes threshold
   - fallback prompt otherwise
4. Start model streaming.
5. For each model chunk:
   - append to an in-memory buffer,
   - emit a `chunk` SSE event.
6. When model generation finishes:
   - persist one final assistant message using the full buffered answer,
   - update conversation `updated_at`,
   - emit `done`.

This avoids writing partial assistant text to the database on every token.

## LLM Layer Changes
The current LLM wrapper only exposes full-response generation. Add a streaming method to the LLM abstraction.

### Current interface
`backend/internal/pkg/eino/llm.go`

### New capability
Add something like:

```go
Stream(messages []ChatMessage, onChunk func(string) error) error
```

The implementation should wrap the underlying Ark/Eino streaming interface if supported. The callback contract should be simple:
- called with every non-empty text chunk,
- returns an error if streaming should stop.

Do not redesign the whole LLM client. Keep the existing synchronous `Chat()` method for compatibility.

## RAG Chain Changes
The current chain already decides between RAG and fallback prompt building. Extend it so the chain can support both:
- non-stream execution,
- stream execution.

Recommended structure:
- keep `Execute(ctx, question)` for non-stream callers,
- add `ExecuteStream(ctx, question, onChunk)` for stream callers.

Both should share the same routing logic:
1. retrieve docs,
2. decide RAG vs fallback,
3. build messages,
4. call either normal generate or stream generate.

This avoids duplicating the routing decision in the service layer.

## Prompt Design
You asked that prompts be specific. We will keep that requirement here too.

### RAG streaming prompt
When retrieval succeeds, use the same strong grounded prompt principles:
- prioritize provided article excerpts,
- summarize and reorganize clearly,
- do not invent blog claims,
- if partial coverage only, say exactly what the site content covers.

### Fallback streaming prompt
When retrieval is weak or empty:
- explicitly say that the answer is based on general knowledge,
- do not pretend it comes from site articles,
- answer concretely and practically,
- mention assumptions when the question is ambiguous.

Streaming does not change prompt semantics. It only changes how the final answer is delivered to the frontend.

## Conversation Rename Design
### Backend
Add a new request DTO for rename:

```go
type UpdateConversationRequest struct {
    Title string `json:"title" binding:"required,min=1,max=255"`
}
```

Add handler method:
- fetch conversation by id,
- verify ownership,
- update title,
- save,
- return updated conversation.

### Frontend
Add lightweight inline rename behavior:
- current conversation title in the header can enter edit mode,
- optionally also allow rename from the sidebar item,
- `Enter` saves,
- `Escape` cancels,
- blur can save if the content changed.

Recommendation: implement rename first in the main header only. That keeps the UI focused and avoids duplicating rename state in sidebar items.

## Sorting Rules
The current sorting rule is correct and should remain unchanged:
- empty conversations first,
- non-empty conversations by `updatedAt` descending.

Important state rule during streaming:
- a conversation that was empty becomes non-empty as soon as the first user message is added,
- once `updatedAt` changes during send/stream completion, the conversation should move naturally according to the existing sort comparator.

No new sorting mode is needed.

## Frontend Store Design
`frontend/src/store/chatStore.ts` will need a streaming-aware send path.

### New behavior for `sendMessage`
1. Ensure there is an active conversation.
2. Append the user message locally immediately.
3. Insert a placeholder assistant message with empty content.
4. Open the SSE stream.
5. On every `chunk` event:
   - append the chunk text to the placeholder assistant message.
6. On `done`:
   - clear typing/streaming state,
   - optionally refresh conversation detail if needed.
7. On `error`:
   - replace placeholder text with an error message or mark it failed,
   - clear typing state.

### State additions
The store may need a small amount of new state, for example:
- `isStreaming`
- `streamingConversationId`
- `renamingConversationId` is **not** needed in the store if rename UI state lives in the component.

Keep UI-only transient rename state inside `AIChat.tsx`, not in global Zustand state.

## Frontend API Layer
`frontend/src/api/chat.ts` should add:
- `streamMessage(...)` using raw `fetch` or `EventSource`-style handling.
- `renameConversation(id, title)` for `PATCH /api/chat/conversations/:id`.

Because native `EventSource` only supports GET, the easiest approach is usually `fetch()` + `ReadableStream` parsing for POST-based SSE. This gives full control over auth headers and request body.

Recommended choice:
- use `fetch()` with `Authorization` header,
- parse the SSE stream manually from `response.body`.

This avoids having to redesign auth around query params or cookies.

## Auto-Scroll Design
We should not always force-scroll. The behavior should be conditional.

### Rule
If the user is already near the bottom of the message container, auto-scroll as new chunks arrive.
If the user has scrolled upward intentionally, do not yank the view back down.

### Suggested threshold
Treat “near bottom” as within `80px` of the bottom.

### Implementation shape
In `AIChat.tsx`:
- attach a ref to the scroll container,
- on scroll, compute whether the user is near bottom,
- when messages or assistant content changes, scroll to bottom only if near bottom is true.

This gives the expected chat-app behavior.

## Error Handling
### Streaming start failure
If the SSE request fails before any content is received:
- stop typing state,
- show toast or assistant error bubble.

### Mid-stream failure
If some chunks already arrived and then the stream breaks:
- keep the partial content,
- append a short note such as “生成中断，请稍后重试” or mark the bubble failed,
- do not silently discard the partial answer.

### Persistence failure after stream end
If the final assistant message cannot be written to MySQL:
- still return the streamed content to the user,
- surface a warning in logs,
- optionally refetch later or mark sync issue.

The user experience should not collapse just because final persistence fails.

## Testing Plan
### Backend tests
1. streaming endpoint writes user message immediately
2. streaming endpoint accumulates assistant chunks and persists one final assistant message
3. rename endpoint rejects non-owner
4. rename endpoint accepts valid owner request
5. fallback prompt still works during streaming mode
6. RAG prompt still works during streaming mode

### Frontend tests or verification points
1. user sees assistant text appear progressively instead of all at once
2. rename updates the visible conversation title immediately
3. empty conversation stays first until it becomes non-empty
4. non-empty conversation moves by latest update time
5. auto-scroll follows streamed output when already near bottom
6. auto-scroll does not force-jump when user has scrolled upward

## Backward Compatibility
- Existing conversation persistence schema remains unchanged.
- Existing non-stream `POST /api/ai/chat` can stay available.
- Existing frontend chat history APIs remain valid.
- Sorting logic is preserved.

## Acceptance Criteria
This design is successful when:
- assistant replies stream progressively in the chat UI,
- users can rename conversations,
- conversation ordering remains: empty first, non-empty newest first,
- the viewport follows assistant output while the user is near the bottom,
- the user no longer has to manually scroll during long streamed replies,
- conversation history persistence still works correctly in MySQL.
