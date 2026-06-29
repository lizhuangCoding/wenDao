# App Task Runner Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace scattered in-process fire-and-forget goroutines with an application-level task runner that provides lifecycle control, timeout, retry, and in-memory metrics while leaving durable retries to `async_jobs`.

**Architecture:** Introduce a shared `TaskRunner` in `internal/pkg/async` and thread it into server bootstrap, handlers, and selected services. Long-lived schedulers and short-lived background tasks will submit work to this runner; process shutdown will cancel and wait on outstanding tasks. Tasks that require cross-restart durability will continue to use the DB-backed async job system.

**Tech Stack:** Go, context, sync.WaitGroup, atomic counters, Zap logger, existing server bootstrap

---

### Task 1: Build the task runner primitive

**Files:**
- Create: `backend/internal/pkg/async/task_runner.go`
- Modify: `backend/internal/pkg/async/async.go`
- Modify: `backend/internal/pkg/async/async_test.go`
- Create: `backend/internal/pkg/async/task_runner_test.go`

- [ ] Add failing tests for task submission, retry, timeout, shutdown wait, and stats snapshot.
- [ ] Implement `TaskRunner`, task options, merged context behavior, shutdown semantics, and in-memory metrics.
- [ ] Keep a minimal compatibility wrapper in `async.go` only for legacy paths that are still outside this scope.
- [ ] Re-run async package tests and keep them green.

### Task 2: Wire the runner through server lifecycle and handlers

**Files:**
- Modify: `backend/cmd/server/app.go`
- Modify: `backend/cmd/server/background_tasks.go`
- Modify: `backend/cmd/server/background_tasks_test.go`
- Modify: `backend/cmd/server/bootstrap_http.go`
- Modify: `backend/cmd/server/bootstrap_services.go`
- Modify: `backend/internal/handler/article/article.go`
- Modify: `backend/internal/handler/comment/comment.go`
- Modify: `backend/internal/handler/facade.go`

- [ ] Add failing tests covering scheduler startup on the runner and handler-triggered background stat work through the runner.
- [ ] Create one shared app runner in bootstrap, store it in `appServices`, and use it for all recurring background tasks.
- [ ] Stop using direct `async.Go` in handlers; submit bounded tasks to the runner instead.
- [ ] Re-run focused server and handler tests and keep them green.

### Task 3: Move selected service-side fire-and-forget work onto the runner

**Files:**
- Modify: `backend/internal/service/article/article_service.go`
- Modify: `backend/internal/service/article/article_read.go`
- Modify: `backend/internal/service/article/article_cache_test.go`
- Modify: `backend/internal/service/facade.go`

- [ ] Add failing tests for article cache warmup being submitted through the runner instead of bare goroutines.
- [ ] Thread the runner into `ArticleService` as an option and use it for cache warmup tasks.
- [ ] Leave durable vector/notification work on the DB async job path; do not collapse the two systems together.
- [ ] Re-run focused service tests and then broader backend verification.
