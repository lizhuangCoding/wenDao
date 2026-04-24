# Automatic Article Vector Indexing Design

## Overview
Remove the manual “同步 AI 索引” admin action and make automatic asynchronous Redis vector indexing the only supported indexing mechanism.

The current codebase already vectorizes articles automatically on publish and on updates to published articles. This design removes the remaining manual trigger entry points so the behavior matches the intended product model:
- publishing a published article or changing a published article updates Redis vectors automatically,
- drafts do not enter the vector index,
- draft/delete actions remove vectors automatically,
- administrators no longer need or see a manual sync control.

## Goal
Ensure article vector indexing happens only as a side effect of article lifecycle events, not through a separate manual admin button or endpoint.

## Current State
### Already automatic today
The backend already performs asynchronous vector operations in the article service:
- create published article → async vectorize
- update published article → async re-vectorize
- publish draft article → async vectorize
- draft article → async delete vectors
- delete article → async delete vectors

### Still manual today
There is still a manual admin control path:
- frontend button in `frontend/src/views/admin/articles/ArticleList.tsx`
- backend handler in `backend/internal/handler/article.go`
- backend route in `backend/cmd/server/main.go`
- backend service method `RevectorizeAll()` in `backend/internal/service/article.go`

This creates mixed UX: the system is already mostly automatic, but the UI still suggests manual sync is necessary.

## Design Summary
Use the existing automatic asynchronous vector indexing logic as the only indexing path.

Remove:
- the admin button for manual AI sync,
- the manual admin API endpoint,
- the service method used only by that endpoint.

Keep:
- automatic async vectorization on publish,
- automatic async re-vectorization when editing a published article,
- automatic async vector deletion when drafting or deleting an article.

## Files Affected
### Frontend
- Modify: `frontend/src/views/admin/articles/ArticleList.tsx`

### Backend
- Modify: `backend/internal/handler/article.go`
- Modify: `backend/internal/service/article.go`
- Modify: `backend/cmd/server/main.go`

## Lifecycle Rules
This design makes the article/vector relationship explicit.

### 1. Create draft article
- no vectorization

### 2. Create published article
- automatically enqueue async vectorization

### 3. Update published article
- automatically enqueue async re-vectorization using the new title/content

### 4. Update draft article
- no vectorization

### 5. Publish draft article
- automatically enqueue async vectorization

### 6. Convert published article to draft
- automatically enqueue async vector deletion

### 7. Delete article
- automatically enqueue async vector deletion

These rules already mostly exist in the current `articleService`. The design is mainly about removing the inconsistent manual path.

## Frontend Design
In `frontend/src/views/admin/articles/ArticleList.tsx`:
- remove `revectorizeMutation`
- remove the “同步 AI 索引” button from the action bar
- keep the rest of the article management actions unchanged

The article list should continue to offer only content-management operations:
- create
- publish/draft toggle
- edit
- delete

This keeps admin UX aligned with the actual system behavior.

## Backend Design
### Handler layer
In `backend/internal/handler/article.go`:
- remove `RevectorizeAll` handler

### Service layer
In `backend/internal/service/article.go`:
- remove `RevectorizeAll()` from `ArticleService`
- remove `articleService.RevectorizeAll()` implementation

### Routing
In `backend/cmd/server/main.go`:
- remove the admin route for `/admin/articles/revectorize`

## Why Remove the Manual Path
The system already does the correct thing automatically in the main article lifecycle.
Keeping the manual path has several downsides:
- it implies to admins that syncing is a required extra step,
- it duplicates responsibility,
- it makes failures and expected behavior harder to reason about,
- it increases backend maintenance surface for little value.

Removing it simplifies the product model and the codebase.

## Error Handling
Automatic vector operations remain asynchronous and non-blocking.

This means:
- article create/update/publish should still succeed even if vectorization fails,
- vectorization errors should continue to be logged,
- user-facing article management should not be blocked by vector index failures.

This is important because vector indexing is a secondary asynchronous concern, not the primary article publishing transaction.

## Logging
Keep the existing logging behavior around async vectorization failures.
That is sufficient for now:
- no extra UI feedback is required,
- no new notification system is needed in this change.

## Testing Plan
### Frontend verification
1. Open the admin article list.
2. Confirm the “同步 AI 索引” button is no longer visible.
3. Confirm the rest of the actions still render and work.

### Backend verification
1. Create a published article and confirm no manual sync endpoint is needed.
2. Update a published article and confirm the code path still triggers async vectorization.
3. Publish a draft article and confirm async vectorization still triggers.
4. Convert a published article to draft and confirm vector deletion logic still remains.
5. Delete an article and confirm vector deletion logic still remains.
6. Confirm the removed route no longer exists.

## Acceptance Criteria
This design is successful when:
- the admin article list no longer shows a manual AI sync button,
- the backend no longer exposes a manual full re-vectorization endpoint,
- published article creation still triggers async vectorization,
- published article updates still trigger async re-vectorization,
- drafting or deleting articles still removes vectors asynchronously,
- admins no longer need to think about manual vector index syncing.
