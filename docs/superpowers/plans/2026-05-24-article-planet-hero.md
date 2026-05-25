# Article Planet Hero Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an immersive homepage 3D article planet where every published article is represented as an interactive node that links to `/article/:slug`.

**Architecture:** Add a lightweight public orbit endpoint for all published articles, then keep the 3D work isolated in focused frontend home components. Use a tested pure layout module for stable article-to-sphere mapping, and load the Three.js scene lazily so non-home routes do not pay the 3D bundle cost.

**Tech Stack:** Go, Gin, GORM, React 18, TypeScript, TanStack Query, Tailwind CSS, `three`, `@react-three/fiber`, `@react-three/drei`, Node `node:test`.

---

## File Structure

**Create:**
- `/Users/lizhuang/go/src/wenDao/frontend/src/components/home/ArticlePlanetHero.tsx` - first-screen hero shell, loading/error fallback, overlay plus lazy 3D scene.
- `/Users/lizhuang/go/src/wenDao/frontend/src/components/home/ArticlePlanetScene.tsx` - Three/R3F canvas, camera, controls, lights, planet mesh, stars, article nodes.
- `/Users/lizhuang/go/src/wenDao/frontend/src/components/home/ArticlePlanetNode.tsx` - individual article node mesh and pointer interactions.
- `/Users/lizhuang/go/src/wenDao/frontend/src/components/home/ArticlePlanetOverlay.tsx` - DOM slogan/search/category/selected article overlay.
- `/Users/lizhuang/go/src/wenDao/frontend/src/components/home/articlePlanetLayout.ts` - pure article-to-sphere layout and weight/color helpers.
- `/Users/lizhuang/go/src/wenDao/frontend/src/components/home/articlePlanetLayout.test.mjs` - Node tests for layout determinism, weight, colors, and empty input.
- `/Users/lizhuang/go/src/wenDao/frontend/src/components/home/index.ts` - home component exports.

**Modify:**
- `/Users/lizhuang/go/src/wenDao/backend/internal/repository/article/article.go` - add `ListOrbitArticles` repository method with selected columns and category preload.
- `/Users/lizhuang/go/src/wenDao/backend/internal/service/article/article_service.go` - add `ListOrbitArticles` to the article service interface.
- `/Users/lizhuang/go/src/wenDao/backend/internal/service/article/article_read.go` - implement service pass-through for orbit articles.
- `/Users/lizhuang/go/src/wenDao/backend/internal/handler/article/article.go` - add orbit response DTOs and public handler.
- `/Users/lizhuang/go/src/wenDao/backend/cmd/server/bootstrap_http.go` - register `GET /api/articles/orbit` before `GET /api/articles/:id`.
- `/Users/lizhuang/go/src/wenDao/backend/cmd/server/routes_test.go` - assert the orbit route is registered.
- `/Users/lizhuang/go/src/wenDao/backend/internal/handler/article/article_access_test.go` - add handler test for orbit response shape.
- `/Users/lizhuang/go/src/wenDao/backend/internal/service/article/article_viewcount_test.go` - update repository stub for new interface method.
- `/Users/lizhuang/go/src/wenDao/backend/internal/service/comment/notification_test.go` - update repository stub for new interface method.
- `/Users/lizhuang/go/src/wenDao/backend/internal/service/knowledge/knowledge_document_test.go` - update repository stub for new interface method.
- `/Users/lizhuang/go/src/wenDao/frontend/src/types/index.ts` - add `ArticleOrbitItem` and `ArticleOrbitResponse` types.
- `/Users/lizhuang/go/src/wenDao/frontend/src/api/article.ts` - add `getArticleOrbit`.
- `/Users/lizhuang/go/src/wenDao/frontend/src/pages/Home.tsx` - replace the current hero/category block with `ArticlePlanetHero`, keep article list and pagination below.
- `/Users/lizhuang/go/src/wenDao/frontend/vite.config.ts` - split Three/R3F dependencies into a `three-vendor` chunk.
- `/Users/lizhuang/go/src/wenDao/frontend/package.json` - add 3D dependencies.
- `/Users/lizhuang/go/src/wenDao/frontend/package-lock.json` - update lockfile from `npm install`.

---

### Task 1: Add Backend Orbit Endpoint Tests

**Files:**
- Modify: `/Users/lizhuang/go/src/wenDao/backend/internal/handler/article/article_access_test.go`
- Modify: `/Users/lizhuang/go/src/wenDao/backend/cmd/server/routes_test.go`

- [ ] **Step 1: Write the failing handler test**

Modify `/Users/lizhuang/go/src/wenDao/backend/internal/handler/article/article_access_test.go`.

Add `time` to the imports:

```go
import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"wenDao/internal/model"
)
```

Add a field to `stubArticleService`:

```go
type stubArticleService struct {
	articleByID      *model.Article
	articleBySlug    *model.Article
	orbitArticles    []*model.Article
	incrViewCountIDs []int64
	listPage         int
	listPageSize     int
	batchIDs         []int64
}
```

Add this method to `stubArticleService`:

```go
func (s *stubArticleService) ListOrbitArticles() ([]*model.Article, error) {
	return s.orbitArticles, nil
}
```

Append this test:

```go
func TestArticleHandlerListOrbitArticles_ReturnsLightweightArticleNodes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	createdAt := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	coverImage := "/uploads/cover.png"
	articleSvc := &stubArticleService{
		orbitArticles: []*model.Article{
			{
				ID:           7,
				Title:        "3D 星球",
				Slug:         "planet",
				Summary:      "首屏文章星球",
				Content:      "正文不应该出现在 orbit 响应中",
				CoverImage:   &coverImage,
				Status:       "published",
				SourceType:   model.ArticleSourceTypeKnowledgeDocument,
				ViewCount:    120,
				CommentCount: 5,
				IsTop:        true,
				CreatedAt:    createdAt,
				Category: &model.Category{
					ID:   3,
					Name: "AI",
					Slug: "ai",
				},
			},
		},
	}
	h := NewArticleHandler(articleSvc, nil, &stubSettingService{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/articles/orbit", nil)

	h.ListOrbitArticles(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d and body %s", w.Code, w.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	data := body["data"].(map[string]any)
	items := data["data"].([]any)
	if data["total"].(float64) != 1 {
		t.Fatalf("expected total 1, got %#v", data["total"])
	}
	item := items[0].(map[string]any)
	if item["title"] != "3D 星球" || item["slug"] != "planet" {
		t.Fatalf("expected article identity fields, got %#v", item)
	}
	if _, ok := item["content"]; ok {
		t.Fatalf("expected orbit response to omit content, got %#v", item)
	}
	category := item["category"].(map[string]any)
	if category["name"] != "AI" || category["slug"] != "ai" {
		t.Fatalf("expected category summary, got %#v", category)
	}
}
```

- [ ] **Step 2: Add the route expectation**

Modify `/Users/lizhuang/go/src/wenDao/backend/cmd/server/routes_test.go`.

Add this entry to the `required` slice, immediately after `"GET /api/articles"`:

```go
"GET /api/articles/orbit",
```

- [ ] **Step 3: Run the focused tests and verify they fail**

Run:

```bash
cd /Users/lizhuang/go/src/wenDao/backend
env GOCACHE=/private/tmp/wendao-go-cache GOTOOLCHAIN=go1.25.3 go test ./internal/handler/article ./cmd/server -run 'TestArticleHandlerListOrbitArticles|TestBuildRouter_RegistersRequiredRoutes' -count=1
```

Expected: FAIL because `ArticleHandler.ListOrbitArticles` and the route are not implemented yet.

---

### Task 2: Implement Backend Orbit Endpoint

**Files:**
- Modify: `/Users/lizhuang/go/src/wenDao/backend/internal/repository/article/article.go`
- Modify: `/Users/lizhuang/go/src/wenDao/backend/internal/service/article/article_service.go`
- Modify: `/Users/lizhuang/go/src/wenDao/backend/internal/service/article/article_read.go`
- Modify: `/Users/lizhuang/go/src/wenDao/backend/internal/handler/article/article.go`
- Modify: `/Users/lizhuang/go/src/wenDao/backend/cmd/server/bootstrap_http.go`
- Modify: `/Users/lizhuang/go/src/wenDao/backend/internal/service/article/article_viewcount_test.go`
- Modify: `/Users/lizhuang/go/src/wenDao/backend/internal/service/comment/notification_test.go`
- Modify: `/Users/lizhuang/go/src/wenDao/backend/internal/service/knowledge/knowledge_document_test.go`
- Test: `/Users/lizhuang/go/src/wenDao/backend/internal/handler/article/article_access_test.go`
- Test: `/Users/lizhuang/go/src/wenDao/backend/cmd/server/routes_test.go`

- [ ] **Step 1: Add repository interface and implementation**

Modify `/Users/lizhuang/go/src/wenDao/backend/internal/repository/article/article.go`.

Add this method to `ArticleRepository`:

```go
	ListOrbitArticles() ([]*model.Article, error)
```

Add this implementation near `List`:

```go
// ListOrbitArticles 获取首页文章星球需要的轻量文章数据。
func (r *articleRepository) ListOrbitArticles() ([]*model.Article, error) {
	var articles []*model.Article
	err := r.db.Model(&model.Article{}).
		Select("id", "title", "slug", "summary", "cover_image", "status", "source_type", "view_count", "comment_count", "is_top", "category_id", "published_at", "created_at").
		Where("status = ?", "published").
		Preload("Category", func(db *gorm.DB) *gorm.DB {
			return db.Select("id", "name", "slug")
		}).
		Order("is_top DESC, published_at DESC, created_at DESC").
		Find(&articles).Error
	return articles, err
}
```

- [ ] **Step 2: Add service interface and implementation**

Modify `/Users/lizhuang/go/src/wenDao/backend/internal/service/article/article_service.go`.

Add this method to `ArticleService`:

```go
	ListOrbitArticles() ([]*model.Article, error)
```

Modify `/Users/lizhuang/go/src/wenDao/backend/internal/service/article/article_read.go`.

Add:

```go
// ListOrbitArticles 获取首页文章星球需要的轻量文章数据。
func (s *articleService) ListOrbitArticles() ([]*model.Article, error) {
	articles, err := s.articleRepo.ListOrbitArticles()
	if err != nil {
		return nil, fmt.Errorf("failed to list orbit articles: %w", err)
	}
	return articles, nil
}
```

- [ ] **Step 3: Add handler DTOs and handler method**

Modify `/Users/lizhuang/go/src/wenDao/backend/internal/handler/article/article.go`.

Add these DTOs after the request structs:

```go
type ArticleOrbitCategory struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type ArticleOrbitItem struct {
	ID           int64                 `json:"id"`
	Title        string                `json:"title"`
	Slug         string                `json:"slug"`
	Summary      string                `json:"summary"`
	CoverImage   *string               `json:"cover_image,omitempty"`
	ViewCount    int                   `json:"view_count"`
	CommentCount int                   `json:"comment_count"`
	IsTop        bool                  `json:"is_top"`
	SourceType   string                `json:"source_type"`
	Category     *ArticleOrbitCategory `json:"category,omitempty"`
	CreatedAt    string                `json:"created_at"`
}

func toArticleOrbitItem(article *model.Article) ArticleOrbitItem {
	item := ArticleOrbitItem{
		ID:           article.ID,
		Title:        article.Title,
		Slug:         article.Slug,
		Summary:      article.Summary,
		CoverImage:   article.CoverImage,
		ViewCount:    article.ViewCount,
		CommentCount: article.CommentCount,
		IsTop:        article.IsTop,
		SourceType:   article.SourceType,
		CreatedAt:    article.CreatedAt.Format(time.RFC3339),
	}
	if article.Category != nil {
		item.Category = &ArticleOrbitCategory{
			ID:   article.Category.ID,
			Name: article.Category.Name,
			Slug: article.Category.Slug,
		}
	}
	return item
}
```

Add `time` and `wenDao/internal/model` to the imports if they are not already present:

```go
import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"wenDao/internal/model"
	"wenDao/internal/pkg/pagination"
	"wenDao/internal/pkg/response"
	"wenDao/internal/service"
)
```

Add this handler near `List`:

```go
// ListOrbitArticles 获取首页 3D 文章星球所需的轻量文章列表。
func (h *ArticleHandler) ListOrbitArticles(c *gin.Context) {
	articles, err := h.articleService.ListOrbitArticles()
	if err != nil {
		response.InternalErrorWithErr(c, "Failed to list orbit articles", err)
		return
	}

	items := make([]ArticleOrbitItem, 0, len(articles))
	for _, article := range articles {
		if article == nil {
			continue
		}
		items = append(items, toArticleOrbitItem(article))
	}

	response.Success(c, gin.H{
		"data":  items,
		"total": len(items),
	})
}
```

- [ ] **Step 4: Register route before the `:id` route**

Modify `/Users/lizhuang/go/src/wenDao/backend/cmd/server/bootstrap_http.go`.

Change the public article routes to:

```go
api.GET("/articles", articleHandler.List)
api.GET("/articles/orbit", articleHandler.ListOrbitArticles)
api.GET("/articles/:id", articleHandler.GetByID)
api.GET("/articles/slug/:slug", articleHandler.GetBySlug)
```

- [ ] **Step 5: Update repository test stubs for the new interface**

Add this method to each ArticleRepository stub listed below:

```go
func (r *viewCountArticleRepoStub) ListOrbitArticles() ([]*model.Article, error) {
	return nil, nil
}
```

File:
- `/Users/lizhuang/go/src/wenDao/backend/internal/service/article/article_viewcount_test.go`

Add this method:

```go
func (r *replyNotificationArticleRepo) ListOrbitArticles() ([]*model.Article, error) {
	return nil, nil
}
```

File:
- `/Users/lizhuang/go/src/wenDao/backend/internal/service/comment/notification_test.go`

Add this method:

```go
func (r *stubKnowledgeArticleRepository) ListOrbitArticles() ([]*model.Article, error) {
	return nil, nil
}
```

File:
- `/Users/lizhuang/go/src/wenDao/backend/internal/service/knowledge/knowledge_document_test.go`

- [ ] **Step 6: Format and run focused backend tests**

Run:

```bash
cd /Users/lizhuang/go/src/wenDao/backend
gofmt -w internal/repository/article/article.go internal/service/article/article_service.go internal/service/article/article_read.go internal/handler/article/article.go cmd/server/bootstrap_http.go cmd/server/routes_test.go internal/handler/article/article_access_test.go internal/service/article/article_viewcount_test.go internal/service/comment/notification_test.go internal/service/knowledge/knowledge_document_test.go
env GOCACHE=/private/tmp/wendao-go-cache GOTOOLCHAIN=go1.25.3 go test ./internal/handler/article ./cmd/server ./internal/service/article ./internal/service/comment ./internal/service/knowledge -run 'TestArticleHandlerListOrbitArticles|TestBuildRouter_RegistersRequiredRoutes|Test' -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit backend orbit endpoint**

Run:

```bash
cd /Users/lizhuang/go/src/wenDao
git add backend/internal/repository/article/article.go backend/internal/service/article/article_service.go backend/internal/service/article/article_read.go backend/internal/handler/article/article.go backend/cmd/server/bootstrap_http.go backend/cmd/server/routes_test.go backend/internal/handler/article/article_access_test.go backend/internal/service/article/article_viewcount_test.go backend/internal/service/comment/notification_test.go backend/internal/service/knowledge/knowledge_document_test.go
git commit -m "feat: add article orbit endpoint"
```

---

### Task 3: Add Frontend Article Planet Layout Tests

**Files:**
- Create: `/Users/lizhuang/go/src/wenDao/frontend/src/components/home/articlePlanetLayout.test.mjs`

- [ ] **Step 1: Write the failing layout tests**

Create `/Users/lizhuang/go/src/wenDao/frontend/src/components/home/articlePlanetLayout.test.mjs`:

```js
import assert from 'node:assert/strict';
import { rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import test from 'node:test';
import { build } from 'esbuild';

const tempDir = path.join(tmpdir(), 'wendao-article-planet-layout-tests');
const bundlePath = path.join(tempDir, 'articlePlanetLayout.test-bundle.mjs');

const loadLayout = async () => {
  await build({
    entryPoints: [new URL('./articlePlanetLayout.ts', import.meta.url).pathname],
    bundle: true,
    format: 'esm',
    outfile: bundlePath,
    platform: 'node',
  });

  return import(`file://${bundlePath}?cache=${Date.now()}`);
};

const makeArticle = (overrides = {}) => ({
  id: 1,
  title: '文章',
  slug: 'article',
  summary: '摘要',
  view_count: 0,
  comment_count: 0,
  is_top: false,
  source_type: 'manual',
  category: { id: 1, name: 'AI', slug: 'ai' },
  created_at: '2026-05-24T12:00:00Z',
  ...overrides,
});

test.after(async () => {
  await rm(tempDir, { recursive: true, force: true });
});

test('buildArticlePlanetLayout returns one stable node per article', async () => {
  const { buildArticlePlanetLayout } = await loadLayout();
  const articles = [
    makeArticle({ id: 1, slug: 'a' }),
    makeArticle({ id: 2, slug: 'b', category: { id: 2, name: 'Go', slug: 'go' } }),
    makeArticle({ id: 3, slug: 'c' }),
  ];

  const first = buildArticlePlanetLayout(articles);
  const second = buildArticlePlanetLayout(articles);

  assert.equal(first.length, 3);
  assert.deepEqual(second, first);
  for (const node of first) {
    const distance = Math.hypot(node.position[0], node.position[1], node.position[2]);
    assert.ok(distance > 2.35 && distance < 2.75, `expected node on sphere surface, got ${distance}`);
  }
});

test('calculateArticlePlanetWeight rewards top and active articles', async () => {
  const { calculateArticlePlanetWeight } = await loadLayout();

  const base = calculateArticlePlanetWeight(makeArticle());
  const active = calculateArticlePlanetWeight(makeArticle({ is_top: true, view_count: 1000, comment_count: 25 }));

  assert.ok(active > base);
  assert.ok(active <= 3);
});

test('getArticlePlanetColor maps the same category to the same color', async () => {
  const { getArticlePlanetColor } = await loadLayout();

  assert.equal(getArticlePlanetColor(4), getArticlePlanetColor(4));
  assert.notEqual(getArticlePlanetColor(4), getArticlePlanetColor(5));
});

test('buildArticlePlanetLayout handles empty article lists', async () => {
  const { buildArticlePlanetLayout } = await loadLayout();

  assert.deepEqual(buildArticlePlanetLayout([]), []);
});
```

- [ ] **Step 2: Run the test and verify it fails**

Run:

```bash
cd /Users/lizhuang/go/src/wenDao/frontend
node --test src/components/home/articlePlanetLayout.test.mjs
```

Expected: FAIL because `articlePlanetLayout.ts` does not exist yet.

---

### Task 4: Implement Frontend Orbit Types, API, and Layout Logic

**Files:**
- Modify: `/Users/lizhuang/go/src/wenDao/frontend/src/types/index.ts`
- Modify: `/Users/lizhuang/go/src/wenDao/frontend/src/api/article.ts`
- Create: `/Users/lizhuang/go/src/wenDao/frontend/src/components/home/articlePlanetLayout.ts`
- Test: `/Users/lizhuang/go/src/wenDao/frontend/src/components/home/articlePlanetLayout.test.mjs`

- [ ] **Step 1: Add orbit article types**

Modify `/Users/lizhuang/go/src/wenDao/frontend/src/types/index.ts`.

Add after `ArticleListItem`:

```ts
export interface ArticleOrbitCategory {
  id: number;
  name: string;
  slug: string;
}

export interface ArticleOrbitItem {
  id: number;
  title: string;
  slug: string;
  summary: string;
  cover_image?: string;
  view_count: number;
  comment_count: number;
  is_top: boolean;
  source_type: 'manual' | 'knowledge_document';
  category?: ArticleOrbitCategory;
  created_at: string;
}

export interface ArticleOrbitResponse {
  data: ArticleOrbitItem[];
  total: number;
}
```

- [ ] **Step 2: Add frontend API method**

Modify `/Users/lizhuang/go/src/wenDao/frontend/src/api/article.ts`.

Add `ArticleOrbitResponse` to the type import:

```ts
  ArticleOrbitResponse,
```

Add this method after `getArticles`:

```ts
  getArticleOrbit: () => {
    return request.get<ArticleOrbitResponse>('/articles/orbit');
  },
```

- [ ] **Step 3: Implement layout helper**

Create `/Users/lizhuang/go/src/wenDao/frontend/src/components/home/articlePlanetLayout.ts`:

```ts
import type { ArticleOrbitItem } from '@/types';

const CATEGORY_COLORS = [
  '#10b981',
  '#38bdf8',
  '#f59e0b',
  '#ec4899',
  '#a78bfa',
  '#f43f5e',
  '#22c55e',
  '#06b6d4',
];

export interface ArticlePlanetNodeLayout {
  article: ArticleOrbitItem;
  color: string;
  emissiveIntensity: number;
  key: string;
  position: [number, number, number];
  radius: number;
  weight: number;
}

const clamp = (value: number, min: number, max: number) => Math.min(max, Math.max(min, value));

export const getArticlePlanetColor = (categoryId?: number) => {
  if (!categoryId || Number.isNaN(categoryId)) {
    return CATEGORY_COLORS[0];
  }
  return CATEGORY_COLORS[Math.abs(categoryId) % CATEGORY_COLORS.length];
};

export const calculateArticlePlanetWeight = (article: ArticleOrbitItem) => {
  const topBonus = article.is_top ? 0.7 : 0;
  const viewBonus = clamp(Math.log10(article.view_count + 1) * 0.25, 0, 0.8);
  const commentBonus = clamp(Math.log10(article.comment_count + 1) * 0.2, 0, 0.5);
  return clamp(1 + topBonus + viewBonus + commentBonus, 1, 3);
};

export const buildArticlePlanetLayout = (
  articles: ArticleOrbitItem[],
  sphereRadius = 2.55
): ArticlePlanetNodeLayout[] => {
  if (articles.length === 0) {
    return [];
  }

  const goldenAngle = Math.PI * (3 - Math.sqrt(5));
  const denominator = Math.max(1, articles.length - 1);

  return articles.map((article, index) => {
    const y = 1 - (index / denominator) * 2;
    const radial = Math.sqrt(Math.max(0, 1 - y * y));
    const theta = index * goldenAngle;
    const categoryOffset = ((article.category?.id ?? 0) % 7) * 0.018;
    const radius = sphereRadius + categoryOffset;
    const weight = calculateArticlePlanetWeight(article);

    return {
      article,
      color: getArticlePlanetColor(article.category?.id),
      emissiveIntensity: 0.55 + weight * 0.28,
      key: `${article.id}-${article.slug}`,
      position: [
        Math.cos(theta) * radial * radius,
        y * radius,
        Math.sin(theta) * radial * radius,
      ],
      radius: 0.035 + weight * 0.018,
      weight,
    };
  });
};
```

- [ ] **Step 4: Run the layout test**

Run:

```bash
cd /Users/lizhuang/go/src/wenDao/frontend
node --test src/components/home/articlePlanetLayout.test.mjs
```

Expected: PASS.

- [ ] **Step 5: Commit frontend data foundation**

Run:

```bash
cd /Users/lizhuang/go/src/wenDao
git add frontend/src/types/index.ts frontend/src/api/article.ts frontend/src/components/home/articlePlanetLayout.ts frontend/src/components/home/articlePlanetLayout.test.mjs
git commit -m "feat: add article planet layout data"
```

---

### Task 5: Add 3D Dependencies and Bundle Split

**Files:**
- Modify: `/Users/lizhuang/go/src/wenDao/frontend/package.json`
- Modify: `/Users/lizhuang/go/src/wenDao/frontend/package-lock.json`
- Modify: `/Users/lizhuang/go/src/wenDao/frontend/vite.config.ts`

- [ ] **Step 1: Install 3D dependencies**

Run:

```bash
cd /Users/lizhuang/go/src/wenDao/frontend
npm install three @react-three/fiber @react-three/drei
```

Expected: `package.json` and `package-lock.json` include the three new runtime dependencies.

- [ ] **Step 2: Add Vite manual chunk for 3D libraries**

Modify `/Users/lizhuang/go/src/wenDao/frontend/vite.config.ts`.

Inside `manualChunks(id)`, add this check before the existing `react-vendor` check:

```ts
            if (
              id.includes('/three/') ||
              id.includes('/@react-three/fiber/') ||
              id.includes('/@react-three/drei/') ||
              id.includes('/maath/') ||
              id.includes('/troika-') ||
              id.includes('/zustand/traditional')
            ) {
              return 'three-vendor'
            }
```

- [ ] **Step 3: Verify dependency install still builds current app**

Run:

```bash
cd /Users/lizhuang/go/src/wenDao/frontend
npm run build
```

Expected: PASS.

- [ ] **Step 4: Commit dependency setup**

Run:

```bash
cd /Users/lizhuang/go/src/wenDao
git add frontend/package.json frontend/package-lock.json frontend/vite.config.ts
git commit -m "chore: add article planet 3d dependencies"
```

---

### Task 6: Build Article Planet Components

**Files:**
- Create: `/Users/lizhuang/go/src/wenDao/frontend/src/components/home/ArticlePlanetNode.tsx`
- Create: `/Users/lizhuang/go/src/wenDao/frontend/src/components/home/ArticlePlanetScene.tsx`
- Create: `/Users/lizhuang/go/src/wenDao/frontend/src/components/home/ArticlePlanetOverlay.tsx`
- Create: `/Users/lizhuang/go/src/wenDao/frontend/src/components/home/ArticlePlanetHero.tsx`
- Create: `/Users/lizhuang/go/src/wenDao/frontend/src/components/home/index.ts`

- [ ] **Step 1: Create the node mesh component**

Create `/Users/lizhuang/go/src/wenDao/frontend/src/components/home/ArticlePlanetNode.tsx`:

```tsx
import type { ThreeEvent } from '@react-three/fiber';
import type { ArticleOrbitItem } from '@/types';
import type { ArticlePlanetNodeLayout } from './articlePlanetLayout';

interface ArticlePlanetNodeProps {
  isActive: boolean;
  node: ArticlePlanetNodeLayout;
  onFocus: (article: ArticleOrbitItem) => void;
  onOpen: (article: ArticleOrbitItem) => void;
}

export const ArticlePlanetNode = ({ isActive, node, onFocus, onOpen }: ArticlePlanetNodeProps) => {
  const handlePointer = (event: ThreeEvent<PointerEvent>) => {
    event.stopPropagation();
    onFocus(node.article);
  };

  const handleClick = (event: ThreeEvent<MouseEvent>) => {
    event.stopPropagation();
    onOpen(node.article);
  };

  return (
    <group position={node.position}>
      <mesh
        onClick={handleClick}
        onPointerOver={handlePointer}
        onPointerMove={handlePointer}
        scale={isActive ? 1.45 : 1}
      >
        <sphereGeometry args={[node.radius, 24, 24]} />
        <meshStandardMaterial
          color={node.color}
          emissive={node.color}
          emissiveIntensity={isActive ? node.emissiveIntensity + 0.7 : node.emissiveIntensity}
          roughness={0.2}
          metalness={0.15}
        />
      </mesh>
      <mesh scale={isActive ? 1.95 : 1.45}>
        <sphereGeometry args={[node.radius, 16, 16]} />
        <meshBasicMaterial color={node.color} transparent opacity={isActive ? 0.22 : 0.12} />
      </mesh>
    </group>
  );
};
```

- [ ] **Step 2: Create the R3F scene**

Create `/Users/lizhuang/go/src/wenDao/frontend/src/components/home/ArticlePlanetScene.tsx`:

```tsx
import { useMemo } from 'react';
import { Canvas } from '@react-three/fiber';
import { AdaptiveDpr, OrbitControls, Preload, Stars } from '@react-three/drei';
import type { ArticleOrbitItem } from '@/types';
import { ArticlePlanetNode } from './ArticlePlanetNode';
import { buildArticlePlanetLayout } from './articlePlanetLayout';

interface ArticlePlanetSceneProps {
  activeArticleId?: number;
  articles: ArticleOrbitItem[];
  onArticleFocus: (article: ArticleOrbitItem) => void;
  onArticleOpen: (article: ArticleOrbitItem) => void;
}

const PlanetBody = () => (
  <group>
    <mesh>
      <sphereGeometry args={[2.18, 72, 72]} />
      <meshPhysicalMaterial
        color="#07111f"
        emissive="#064e3b"
        emissiveIntensity={0.1}
        metalness={0.22}
        opacity={0.42}
        roughness={0.48}
        transparent
        transmission={0.18}
      />
    </mesh>
    <mesh rotation={[Math.PI / 2.2, 0.15, 0]}>
      <torusGeometry args={[2.42, 0.006, 12, 160]} />
      <meshBasicMaterial color="#10b981" transparent opacity={0.32} />
    </mesh>
    <mesh rotation={[Math.PI / 2.8, 0.6, 0.9]}>
      <torusGeometry args={[2.64, 0.004, 12, 160]} />
      <meshBasicMaterial color="#38bdf8" transparent opacity={0.22} />
    </mesh>
  </group>
);

export const ArticlePlanetScene = ({
  activeArticleId,
  articles,
  onArticleFocus,
  onArticleOpen,
}: ArticlePlanetSceneProps) => {
  const nodes = useMemo(() => buildArticlePlanetLayout(articles), [articles]);

  return (
    <Canvas
      camera={{ fov: 45, position: [0, 0, 6.3] }}
      dpr={[1, 1.75]}
      gl={{ antialias: true, alpha: true, powerPreference: 'high-performance' }}
    >
      <color attach="background" args={['#020617']} />
      <ambientLight intensity={0.42} />
      <directionalLight color="#dffdf2" intensity={1.8} position={[4, 3, 5]} />
      <pointLight color="#38bdf8" intensity={22} position={[-3, -1, 3]} />
      <Stars count={900} depth={38} factor={3.2} fade radius={42} speed={0.18} />
      <group rotation={[0.08, -0.24, 0]}>
        <PlanetBody />
        {nodes.map((node) => (
          <ArticlePlanetNode
            key={node.key}
            isActive={node.article.id === activeArticleId}
            node={node}
            onFocus={onArticleFocus}
            onOpen={onArticleOpen}
          />
        ))}
      </group>
      <OrbitControls
        autoRotate
        autoRotateSpeed={0.42}
        enableDamping
        enablePan={false}
        maxDistance={7.2}
        minDistance={4.25}
        rotateSpeed={0.52}
        zoomSpeed={0.45}
      />
      <AdaptiveDpr pixelated />
      <Preload all />
    </Canvas>
  );
};
```

- [ ] **Step 3: Create the DOM overlay**

Create `/Users/lizhuang/go/src/wenDao/frontend/src/components/home/ArticlePlanetOverlay.tsx`:

```tsx
import type { FormEvent } from 'react';
import { ArrowRight, Search, Sparkles } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import type { ArticleOrbitItem, Category } from '@/types';

interface ArticlePlanetOverlayProps {
  activeArticle?: ArticleOrbitItem;
  categories?: Category[];
  inputValue: string;
  selectedCategory?: number;
  slogan?: string;
  onCategoryChange: (categoryId?: number) => void;
  onOpenArticle: (article: ArticleOrbitItem) => void;
  onSearch: (event: FormEvent) => void;
  onSearchInputChange: (value: string) => void;
}

export const ArticlePlanetOverlay = ({
  activeArticle,
  categories,
  inputValue,
  selectedCategory,
  slogan,
  onCategoryChange,
  onOpenArticle,
  onSearch,
  onSearchInputChange,
}: ArticlePlanetOverlayProps) => {
  const { t } = useTranslation();

  return (
    <div className="pointer-events-none absolute inset-0 z-10 flex flex-col justify-end px-6 pb-8 pt-24 sm:px-10 lg:px-12 lg:pb-14">
      <div className="max-w-display mx-auto flex w-full flex-col gap-8 lg:flex-row lg:items-end lg:justify-between">
        <div className="pointer-events-auto max-w-3xl">
          <div className="mb-5 inline-flex items-center gap-3 text-primary-300">
            <Sparkles className="h-4 w-4" />
            <span className="text-xs font-black uppercase tracking-[0.28em]">{t('home.heroSub')}</span>
          </div>
          <h1 className="max-w-4xl text-5xl font-black leading-[1.05] text-white drop-shadow-2xl sm:text-6xl lg:text-7xl">
            {slogan || '我不在执着于得到，而是享受走到'}
          </h1>
          <form onSubmit={onSearch} className="relative mt-8 max-w-xl">
            <input
              type="text"
              placeholder={t('home.searchPlaceholder')}
              className="w-full border-b-2 border-white/25 bg-transparent py-3 pl-0 pr-12 text-sm font-bold tracking-widest text-white outline-none transition-colors placeholder:text-white/45 focus:border-primary-300"
              value={inputValue}
              onChange={(event) => onSearchInputChange(event.target.value)}
            />
            <button
              type="submit"
              className="absolute right-0 top-1/2 inline-flex h-10 w-10 -translate-y-1/2 items-center justify-center text-white/70 transition-colors hover:text-primary-200"
              aria-label={t('home.searchPlaceholder')}
            >
              <Search className="h-5 w-5" />
            </button>
          </form>
          <div className="mt-7 flex gap-3 overflow-x-auto pb-1 scrollbar-hide">
            <button
              type="button"
              onClick={() => onCategoryChange(undefined)}
              className={`shrink-0 border px-4 py-2 text-[10px] font-black uppercase tracking-[0.22em] transition-colors ${
                selectedCategory === undefined
                  ? 'border-primary-300 bg-primary-300 text-neutral-950'
                  : 'border-white/20 bg-white/5 text-white/70 hover:border-white/40 hover:text-white'
              }`}
            >
              {t('home.allArticles')}
            </button>
            {categories?.map((category) => (
              <button
                key={category.id}
                type="button"
                onClick={() => onCategoryChange(category.id)}
                className={`shrink-0 border px-4 py-2 text-[10px] font-black uppercase tracking-[0.22em] transition-colors ${
                  selectedCategory === category.id
                    ? 'border-primary-300 bg-primary-300 text-neutral-950'
                    : 'border-white/20 bg-white/5 text-white/70 hover:border-white/40 hover:text-white'
                }`}
              >
                {category.name}
              </button>
            ))}
          </div>
        </div>

        {activeArticle && (
          <button
            type="button"
            onClick={() => onOpenArticle(activeArticle)}
            className="pointer-events-auto w-full max-w-md border border-white/15 bg-neutral-950/60 p-5 text-left shadow-2xl backdrop-blur-xl transition-colors hover:border-primary-300/70 lg:mb-2"
          >
            <div className="mb-3 flex items-center justify-between gap-4">
              <span className="text-[10px] font-black uppercase tracking-[0.24em] text-primary-300">
                {activeArticle.category?.name || 'Article'}
              </span>
              <ArrowRight className="h-4 w-4 text-white/70" />
            </div>
            <h2 className="text-2xl font-black leading-tight text-white">{activeArticle.title}</h2>
            <p className="mt-3 line-clamp-3 text-sm font-medium leading-relaxed text-white/62">
              {activeArticle.summary || '暂无摘要'}
            </p>
            <div className="mt-5 flex gap-5 text-[10px] font-bold uppercase tracking-widest text-white/45">
              <span>{activeArticle.view_count} views</span>
              <span>{activeArticle.comment_count} comments</span>
            </div>
          </button>
        )}
      </div>
    </div>
  );
};
```

- [ ] **Step 4: Create the hero shell**

Create `/Users/lizhuang/go/src/wenDao/frontend/src/components/home/ArticlePlanetHero.tsx`:

```tsx
import { lazy, Suspense, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type { FormEvent } from 'react';
import { ErrorState, Loading } from '@/components/common';
import type { ArticleOrbitItem, Category } from '@/types';
import { ArticlePlanetOverlay } from './ArticlePlanetOverlay';

const ArticlePlanetScene = lazy(() =>
  import('./ArticlePlanetScene').then((module) => ({ default: module.ArticlePlanetScene }))
);

interface ArticlePlanetHeroProps {
  articles: ArticleOrbitItem[];
  categories?: Category[];
  inputValue: string;
  isError: boolean;
  isLoading: boolean;
  selectedCategory?: number;
  slogan?: string;
  onCategoryChange: (categoryId?: number) => void;
  onSearch: (event: FormEvent) => void;
  onSearchInputChange: (value: string) => void;
}

export const ArticlePlanetHero = ({
  articles,
  categories,
  inputValue,
  isError,
  isLoading,
  selectedCategory,
  slogan,
  onCategoryChange,
  onSearch,
  onSearchInputChange,
}: ArticlePlanetHeroProps) => {
  const navigate = useNavigate();
  const [activeArticleId, setActiveArticleId] = useState<number>();
  const activeArticle = useMemo(
    () => articles.find((article) => article.id === activeArticleId) ?? articles[0],
    [activeArticleId, articles]
  );

  const openArticle = (article: ArticleOrbitItem) => {
    if (article.slug) {
      navigate(`/article/${article.slug}`);
    }
  };

  return (
    <section className="relative -mt-20 min-h-[calc(100vh-1rem)] overflow-hidden bg-neutral-950">
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_68%_45%,rgba(16,185,129,0.28),transparent_30%),radial-gradient(circle_at_30%_85%,rgba(56,189,248,0.18),transparent_28%),linear-gradient(135deg,#020617_0%,#07111f_46%,#030712_100%)]" />
      {isLoading ? (
        <div className="absolute inset-0 flex items-center justify-center">
          <Loading />
        </div>
      ) : isError || articles.length === 0 ? (
        <div className="absolute inset-0 flex items-center justify-center px-6">
          <div className="w-full max-w-md">
            <ErrorState message={isError ? '文章星球加载失败' : '暂无可展示文章'} />
          </div>
        </div>
      ) : (
        <Suspense fallback={<div className="absolute inset-0 flex items-center justify-center"><Loading /></div>}>
          <div className="absolute inset-0 lg:left-[24%]">
            <ArticlePlanetScene
              activeArticleId={activeArticle?.id}
              articles={articles}
              onArticleFocus={(article) => setActiveArticleId(article.id)}
              onArticleOpen={openArticle}
            />
          </div>
        </Suspense>
      )}
      <ArticlePlanetOverlay
        activeArticle={activeArticle}
        categories={categories}
        inputValue={inputValue}
        selectedCategory={selectedCategory}
        slogan={slogan}
        onCategoryChange={onCategoryChange}
        onOpenArticle={openArticle}
        onSearch={onSearch}
        onSearchInputChange={onSearchInputChange}
      />
      <div className="pointer-events-none absolute inset-x-0 bottom-0 h-24 bg-gradient-to-t from-white to-transparent dark:from-neutral-900" />
    </section>
  );
};
```

- [ ] **Step 5: Add home exports**

Create `/Users/lizhuang/go/src/wenDao/frontend/src/components/home/index.ts`:

```ts
export { ArticlePlanetHero } from './ArticlePlanetHero';
```

- [ ] **Step 6: Run frontend type build**

Run:

```bash
cd /Users/lizhuang/go/src/wenDao/frontend
npm run build
```

Expected: PASS with the component files as written in this task.

---

### Task 7: Integrate Article Planet Hero Into Home

**Files:**
- Modify: `/Users/lizhuang/go/src/wenDao/frontend/src/pages/Home.tsx`
- Test: `/Users/lizhuang/go/src/wenDao/frontend/src/components/home/articlePlanetLayout.test.mjs`

- [ ] **Step 1: Update imports**

Modify `/Users/lizhuang/go/src/wenDao/frontend/src/pages/Home.tsx`.

Replace:

```ts
import { Layout, Loading, Pagination, EmptyState, ErrorState } from '@/components/common';
import { ArticleCard } from '@/components/article';
import { motion, AnimatePresence } from 'framer-motion';
```

With:

```ts
import { Layout, Loading, Pagination, EmptyState, ErrorState } from '@/components/common';
import { ArticleCard } from '@/components/article';
import { ArticlePlanetHero } from '@/components/home';
import { motion, AnimatePresence } from 'framer-motion';
```

- [ ] **Step 2: Add orbit query**

In `Home`, after the paginated articles query, add:

```ts
  const {
    data: orbitData,
    isLoading: isOrbitLoading,
    isError: isOrbitError,
  } = useQuery({
    queryKey: ['article-orbit'],
    queryFn: articleApi.getArticleOrbit,
    staleTime: 5 * 60 * 1000,
  });
```

Add this handler after `handleSearch`:

```ts
  const handleCategoryChange = (categoryId?: number) => {
    setSelectedCategory(categoryId);
    setCurrentPage(1);
  };
```

- [ ] **Step 3: Replace the old Hero and Categories sections**

Inside the `return`, replace the current outer page container and the old Hero/Categories markup with this structure:

```tsx
    <Layout>
      <ArticlePlanetHero
        articles={orbitData?.data ?? []}
        categories={categories}
        inputValue={inputValue}
        isError={isOrbitError}
        isLoading={isOrbitLoading}
        selectedCategory={selectedCategory}
        slogan={siteData?.slogan}
        onCategoryChange={handleCategoryChange}
        onSearch={handleSearch}
        onSearchInputChange={setInputValue}
      />

      <div className="max-w-display mx-auto px-6 sm:px-10 lg:px-12 py-24">
        {/* Article Grid */}
        {isLoading ? (
```

Keep the existing article grid, empty state, and pagination markup inside this new container. Delete the old decorative blurred circle and old categories bar, because those controls now live in the hero overlay.

- [ ] **Step 4: Run frontend tests and build**

Run:

```bash
cd /Users/lizhuang/go/src/wenDao/frontend
node --test src/components/home/articlePlanetLayout.test.mjs
npm run build
npm run lint
```

Expected: PASS.

- [ ] **Step 5: Commit home integration**

Run:

```bash
cd /Users/lizhuang/go/src/wenDao
git add frontend/src/components/home frontend/src/pages/Home.tsx
git commit -m "feat: add article planet homepage hero"
```

---

### Task 8: Full Verification and Manual UI Check

**Files:**
- Verify backend and frontend touched files.

- [ ] **Step 1: Run full backend tests**

Run:

```bash
cd /Users/lizhuang/go/src/wenDao/backend
env GOCACHE=/private/tmp/wendao-go-cache GOTOOLCHAIN=go1.25.3 go test ./...
```

Expected: PASS.

- [ ] **Step 2: Run frontend verification**

Run:

```bash
cd /Users/lizhuang/go/src/wenDao/frontend
node --test src/components/home/articlePlanetLayout.test.mjs
npm run build
npm run lint
```

Expected: PASS.

- [ ] **Step 3: Start the frontend dev server**

Run:

```bash
cd /Users/lizhuang/go/src/wenDao/frontend
npm run dev -- --host 127.0.0.1
```

Expected: Vite serves the frontend on `http://127.0.0.1:3000/` or the next available configured port.

- [ ] **Step 4: Manual UI verification**

Open the local frontend and verify:

- The homepage first screen is the 3D article planet.
- The planet rotates automatically.
- Dragging rotates the planet.
- Wheel zoom stays within a controlled range.
- Hovering a node updates the article preview panel.
- Clicking a node opens `/article/:slug`.
- Search and category filters still update the article list below.
- Dark mode keeps the overlay and article list readable.
- Mobile viewport has no text overlap and the planet remains usable.
- If the orbit endpoint is unavailable, the page still shows the fallback and the normal article list.

- [ ] **Step 5: Final commit if verification caused fixes**

If verification required changes, commit them:

```bash
cd /Users/lizhuang/go/src/wenDao
git add backend frontend
git commit -m "fix: polish article planet hero"
```

If no verification fixes were needed, do not create an empty commit.
