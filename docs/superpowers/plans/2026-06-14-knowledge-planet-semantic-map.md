# Knowledge Planet Semantic Map Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a semantic clustering MVP for the homepage knowledge planet by reusing the existing article embedding pipeline without changing RAG chunk retrieval behavior.

**Architecture:** Keep the existing Redis chunk vectors as the RAG retrieval source. Add a persisted article-level semantic profile generated from the same chunk embeddings, including a normalized 3D projection and top semantic neighbors for the orbit API. The frontend uses semantic coordinates when present and keeps the existing golden-angle layout as a fallback.

**Tech Stack:** Go, GORM, Redis Vector, existing Eino embedding adapter, React, Three.js, TypeScript.

---

### Task 1: Persist Article Semantic Profiles

**Files:**
- Create: `backend/internal/model/article_semantic_profile.go`
- Create: `backend/internal/repository/article/article_semantic_profile.go`
- Modify: `backend/internal/pkg/database/migrate.go`
- Modify: `backend/internal/repository/facade.go`
- Test: `backend/internal/service/ai/vector_semantic_profile_test.go`

- [ ] Add a GORM model keyed by `article_id` with `embedding_json`, `map_x`, `map_y`, `map_z`, `content_hash`, `neighbor_json`, and timestamps.
- [ ] Add repository methods `Upsert`, `DeleteByArticleID`, `ListByArticleIDs`, and `ListAll`.
- [ ] Include the model in `AutoMigrate`.

### Task 2: Generate Article-Level Semantics During Vectorization

**Files:**
- Modify: `backend/internal/service/ai/vector.go`
- Modify: `backend/internal/service/facade.go`
- Modify: `backend/cmd/server/bootstrap_infra.go`
- Modify: `backend/cmd/server/bootstrap_services.go`
- Test: `backend/internal/service/ai/vector_semantic_profile_test.go`

- [ ] Extend `VectorService` to accept an optional semantic profile repository.
- [ ] Average normalized chunk embeddings into one normalized article embedding.
- [ ] Project the article embedding to deterministic 3D semantic coordinates.
- [ ] Persist the semantic profile after chunk vectors are written.
- [ ] Delete semantic profiles when article vectors are deleted.

### Task 3: Expose Semantic Map Data In Orbit API

**Files:**
- Modify: `backend/internal/model/article_semantic_profile.go`
- Modify: `backend/internal/service/article/article_service.go`
- Modify: `backend/internal/service/article/article_read.go`
- Modify: `backend/internal/handler/article/article.go`
- Test: `backend/internal/handler/article/article_access_test.go`

- [ ] Hydrate orbit articles with semantic profiles from the article service.
- [ ] Compute top semantic neighbors from article profile cosine similarity.
- [ ] Return `semantic_position` and `semantic_neighbors` from `/api/articles/orbit`.
- [ ] Keep the response valid when semantic profiles are missing.

### Task 4: Render Semantic Clusters On The Planet

**Files:**
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/components/home/articlePlanetLayout.ts`
- Test: `frontend/src/components/home/articlePlanetLayout.test.mjs`

- [ ] Add frontend types for semantic position and neighbors.
- [ ] Update layout to use backend semantic coordinates when present.
- [ ] Add semantic neighbor connections as a new connection strength.
- [ ] Keep category and collection fallback behavior.

### Task 5: Verify And Publish

**Files:**
- All modified files

- [ ] Run `gofmt` on backend Go files.
- [ ] Run `git diff --check`.
- [ ] Run focused Node tests for the planet layout.
- [ ] Run `npm run lint` and `npm run build`.
- [ ] Run backend tests.
- [ ] Merge to `main` and push to GitHub.
