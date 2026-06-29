# Async Job Outbox Consistency Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make comment/article side effects durable by moving cache invalidation, notifications, and vector work into DB-backed async jobs, while wrapping main write paths in transactions where the side effect intent is created.

**Architecture:** Add an `async_jobs` table plus repository/service/worker. Comment creation and comment reaction paths will enqueue jobs inside a transaction with their primary writes. Article vectorization/deletion will stop using fire-and-forget goroutines and instead enqueue durable jobs from transactional article write paths.

**Tech Stack:** Go, GORM, MySQL migrations, Zap logger, existing background task scheduler

---

### Task 1: Define async job persistence and worker boundaries

**Files:**
- Create: `backend/internal/model/async_job.go`
- Create: `backend/internal/repository/asyncjob/async_job.go`
- Create: `backend/internal/service/asyncjob/service.go`
- Create: `backend/internal/service/asyncjob/service_test.go`
- Create: `backend/migrations/000005_add_async_jobs.sql`
- Modify: `backend/internal/pkg/database/migrate.go`
- Modify: `backend/internal/repository/facade.go`
- Modify: `backend/internal/service/facade.go`

- [ ] Add failing tests for async job claim/succeed/fail/retry behavior and run only those tests to confirm the API is missing.
- [ ] Implement the `AsyncJob` model, repository methods (`Enqueue`, `ListRunnable`, `Claim`, `MarkSucceeded`, `MarkFailed`), service-level payload types, and worker dispatch with retry backoff.
- [ ] Register the model in auto-migrate and add the versioned SQL migration for production databases.
- [ ] Re-run the async job tests and keep them green.

### Task 2: Move comment side effects behind transactional async jobs

**Files:**
- Modify: `backend/internal/service/comment/comment.go`
- Modify: `backend/internal/service/comment/comment_test.go`
- Create: `backend/internal/service/comment/transaction.go`
- Modify: `backend/cmd/server/bootstrap_services.go`

- [ ] Add failing tests for comment create success path (comment write + comment count increment + job enqueue), comment create rollback on count-update failure, and comment reaction notification enqueue instead of direct notification creation.
- [ ] Implement a thin comment transaction runner that binds tx-scoped comment/article/job repositories and use it in `Create`, `Like`, `Unlike`, `Dislike`, and `Undislike`.
- [ ] Replace direct cache invalidation / in-app notification / reply email sends with async job enqueue payloads.
- [ ] Re-run focused comment service tests and keep them green.

### Task 3: Replace article fire-and-forget vector work with durable async jobs

**Files:**
- Modify: `backend/internal/service/article/article_service.go`
- Modify: `backend/internal/service/article/article_ai.go`
- Modify: `backend/internal/service/article/article_write.go`
- Modify: `backend/internal/service/article/article_publish.go`
- Create: `backend/internal/service/article/transaction.go`
- Create: `backend/internal/service/article/article_async_jobs_test.go`
- Modify: `backend/cmd/server/background_tasks.go`
- Modify: `backend/cmd/server/bootstrap_infra.go`
- Modify: `backend/cmd/server/bootstrap_services.go`

- [ ] Add failing tests for published article writes enqueuing vector jobs instead of launching goroutines, and for vector-delete paths enqueuing delete jobs.
- [ ] Implement article transaction runner support so published create/update/publish/scheduled-publish paths enqueue vectorize jobs and delete/draft/autosave/delete-batch paths enqueue vector-delete jobs durably.
- [ ] Start an async job background worker from server bootstrap and wire handlers for notification creation, reply email, cache invalidation, vectorize, and vector delete.
- [ ] Re-run focused article/background-task tests, then broader backend verification.
