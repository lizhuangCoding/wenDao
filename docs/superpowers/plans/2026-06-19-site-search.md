# Site Search Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a dedicated public site search page backed by a focused article search API.

**Architecture:** Keep V1 in the existing Go/MySQL and React stack. Add a small search contract to the article repository/service layer, expose it through a new public handler route, and consume it from a new frontend `/search` page that syncs filters with query params.

**Tech Stack:** Go, Gin, GORM, MySQL, React, React Router, TanStack Query, TypeScript, Tailwind, existing common UI primitives.

---

### Task 1: Backend Search Contract

**Files:**
- Modify: `backend/internal/repository/article/article.go`
- Create: `backend/internal/service/article/article_search.go`
- Modify: `backend/internal/service/article/article_service.go`
- Modify: repository stubs in backend tests if compilation requires the new interface methods

- [ ] **Step 1: Add search types and repository method**

Add `ArticleSearchFilter`, `ArticleSearchResult`, and `Search(filter ArticleSearchFilter)` to the article repository package. The filter includes `Keyword`, `CategoryID`, `TagID`, `Page`, and `PageSize`.

- [ ] **Step 2: Implement SQL search query**

Use `articles` as the base table, left join `categories`, `article_tags`, and `tags`, always filter `articles.status = published`, apply optional category/tag filters, and apply keyword matching across `articles.title`, `articles.summary`, `articles.content`, `categories.name`, and `tags.name`. Use `GROUP BY articles.id` to avoid duplicate rows from multiple tags.

- [ ] **Step 3: Add service method and snippet helper**

Add `SearchArticles(keyword string, categoryID, tagID int64, page, pageSize int)` to `ArticleService`. It normalizes pagination, calls the repository, and builds snippets and matched fields from title/summary/content/category/tags.

- [ ] **Step 4: Add focused backend tests**

Add tests for snippet generation and matched fields in `backend/internal/service/article/article_search_test.go`.

### Task 2: Backend HTTP Route

**Files:**
- Modify: `backend/internal/handler/article/article.go`
- Modify: `backend/cmd/server/bootstrap_http.go`
- Modify: `backend/internal/handler/article/article_access_test.go` stubs if required

- [ ] **Step 1: Add handler method**

Add `Search(c *gin.Context)` to `ArticleHandler`. It reads `q`, `category_id`, `tag_id`, and pagination params, calls `articleService.SearchArticles`, and returns the existing paginated response shape.

- [ ] **Step 2: Register public route**

Register `GET /api/search/articles` before authenticated route groups.

- [ ] **Step 3: Ensure route tests compile**

Update service stubs to satisfy the expanded interface and run relevant Go tests.

### Task 3: Frontend API, Types, and Search Page

**Files:**
- Modify: `frontend/src/types/index.ts`
- Create: `frontend/src/api/search.ts`
- Modify: `frontend/src/api/index.ts`
- Create: `frontend/src/pages/Search.tsx`

- [ ] **Step 1: Add frontend types**

Add `ArticleSearchResult` with `article`, `snippet`, and `matched_fields`.

- [ ] **Step 2: Add search API client**

Add `searchApi.searchArticles(params)` calling `/search/articles`.

- [ ] **Step 3: Build `/search` page**

The page reads `q`, `category_id`, `tag_id`, and `page` from query params. It fetches categories/tags for filters and fetches results only when a query or filter exists. It renders loading, error, empty, result list, snippets, and pagination.

### Task 4: Routing and Homepage Integration

**Files:**
- Modify: `frontend/src/router.tsx`
- Modify: `frontend/src/pages/Home.tsx`
- Modify: `frontend/src/router.test.mjs` or add source tests if needed

- [ ] **Step 1: Add route**

Lazy load `Search` and register `/search`.

- [ ] **Step 2: Update homepage search submit**

Change the homepage search form submit to navigate to `/search?q=<input>`. Keep category/tag buttons as homepage list filters.

- [ ] **Step 3: Add source-level frontend tests**

Add or update Node source tests to assert `/search` route exists and homepage search navigates with `q`.

### Task 5: Verification and Commit

**Files:**
- All changed files

- [ ] **Step 1: Format backend**

Run `gofmt -w` on changed Go files.

- [ ] **Step 2: Run backend tests**

Run `env GOTOOLCHAIN=go1.25.3 go test ./...`.

- [ ] **Step 3: Run frontend checks**

Run `npm run lint`, `npm run build`, and `npm run test` from `frontend/`.

- [ ] **Step 4: Commit implementation**

Commit with `feat: add site search page`.
