# Typing ThinkTank CI Splits Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Improve frontend type boundaries, make ThinkTank service options explicit, add CI quality gates, and split targeted oversized modules without changing product behavior.

**Architecture:** Keep changes incremental and behavior-preserving. Add small typed normalizers at API/store boundaries, replace variadic service options with a named options struct, add a GitHub Actions workflow for existing commands, and move focused chat-store helpers into separate modules.

**Tech Stack:** Go, Gin, GORM, TypeScript, React, Zustand, Vite, Node test runner, GitHub Actions.

---

### Task 4: Frontend Type Boundaries

**Files:**
- Modify: `frontend/src/api/client.ts`
- Modify: `frontend/src/api/chat.ts`
- Modify: `frontend/src/store/chatStore.ts`
- Test: `frontend/src/api/client.test.mjs`
- Test: `frontend/src/AIChat.test.mjs`

- [ ] Add tests that assert request helper defaults are not `any`, chat API conversation methods use typed responses, and chat store no longer normalizes steps from `any`.
- [ ] Run `npm run test -- client.test.mjs AIChat.test.mjs` or the closest supported focused node test command and verify the tests fail for the expected source-pattern checks.
- [ ] Replace `any` defaults in `request` with `unknown`, add typed request body generics, and introduce chat conversation response types.
- [ ] Replace chat store `any` normalizer inputs with `unknown` plus guarded object helpers.
- [ ] Run focused frontend tests, `npm run lint`, and `npm run build`.
- [ ] Commit as `refactor: tighten frontend chat typing`.

### Task 5: ThinkTank Explicit Options

**Files:**
- Modify: `backend/internal/service/chat/thinktank_service.go`
- Modify: `backend/internal/service/facade.go`
- Modify: `backend/cmd/server/bootstrap_services.go`
- Modify tests that call `NewThinkTankService`

- [ ] Add or update a Go test that constructs `ThinkTankServiceOptions` and verifies runner-derived dependencies still populate the service.
- [ ] Run the focused Go test and verify it fails before implementation.
- [ ] Replace `options ...any` with `ThinkTankServiceOptions`.
- [ ] Update facade and bootstrap call sites.
- [ ] Run focused chat package tests and backend build.
- [ ] Commit as `refactor: make thinktank options explicit`.

### Task 6: CI Quality Gates

**Files:**
- Create: `.github/workflows/ci.yml`

- [ ] Add a workflow that runs backend Go tests and frontend lint, tests, and build on pushes and pull requests.
- [ ] Validate YAML structure by reading the generated file and running available local commands that mirror CI.
- [ ] Commit as `ci: add backend and frontend quality gates`.

### Task 7: Targeted File Splits

**Files:**
- Create: `frontend/src/store/chatNormalizers.ts`
- Create: `frontend/src/store/chatPersistence.ts`
- Modify: `frontend/src/store/chatStore.ts`
- Test: `frontend/src/AIChat.test.mjs`

- [ ] Add source-pattern tests that require chat normalizers and persistence helpers to live outside `chatStore.ts`.
- [ ] Run focused frontend test and verify it fails before implementation.
- [ ] Move chat step/message normalization helpers into `chatNormalizers.ts`.
- [ ] Move chat localStorage key helpers into `chatPersistence.ts`.
- [ ] Update imports in `chatStore.ts` without changing runtime behavior.
- [ ] Run focused frontend tests, `npm run lint`, and `npm run build`.
- [ ] Commit as `refactor: split chat store helpers`.
