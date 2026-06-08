package article

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"wenDao/internal/model"
	"wenDao/internal/repository"
)

type memoryArticleCacheStore struct {
	mu     sync.Mutex
	values map[string]string
}

func newMemoryArticleCacheStore() *memoryArticleCacheStore {
	return &memoryArticleCacheStore{values: make(map[string]string)}
}

func (s *memoryArticleCacheStore) Get(ctx context.Context, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.values[key]
	if !ok {
		return "", redis.Nil
	}
	return value, nil
}

func (s *memoryArticleCacheStore) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch v := value.(type) {
	case []byte:
		s.values[key] = string(v)
	case string:
		s.values[key] = v
	default:
		payload, err := json.Marshal(v)
		if err != nil {
			return err
		}
		s.values[key] = string(payload)
	}
	return nil
}

func (s *memoryArticleCacheStore) Del(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.values, key)
	return nil
}

func (s *memoryArticleCacheStore) Incr(ctx context.Context, key string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	current := int64(0)
	if raw, ok := s.values[key]; ok && raw != "" {
		if _, err := fmt.Sscanf(raw, "%d", &current); err != nil {
			return 0, err
		}
	}
	current++
	s.values[key] = fmt.Sprintf("%d", current)
	return current, nil
}

type cacheArticleRepoStub struct {
	article        *model.Article
	getByIDCount   int
	getBySlugCount int
	listCount      int
	orbitCount     int
	updateCount    int
}

func (r *cacheArticleRepoStub) Create(article *model.Article) error { return nil }
func (r *cacheArticleRepoStub) GetByID(id int64) (*model.Article, error) {
	r.getByIDCount++
	if r.article == nil {
		return nil, nil
	}
	copied := *r.article
	return &copied, nil
}
func (r *cacheArticleRepoStub) GetBySlug(slug string) (*model.Article, error) {
	r.getBySlugCount++
	if r.article == nil {
		return nil, nil
	}
	copied := *r.article
	return &copied, nil
}
func (r *cacheArticleRepoStub) GetBySource(sourceType string, sourceID int64) (*model.Article, error) {
	return nil, nil
}
func (r *cacheArticleRepoStub) List(filter repository.ArticleFilter) ([]*model.Article, int64, error) {
	r.listCount++
	if r.article == nil {
		return []*model.Article{}, 0, nil
	}
	copied := *r.article
	return []*model.Article{&copied}, 1, nil
}
func (r *cacheArticleRepoStub) ListOrbitArticles() ([]*model.Article, error) {
	r.orbitCount++
	if r.article == nil {
		return []*model.Article{}, nil
	}
	copied := *r.article
	return []*model.Article{&copied}, nil
}
func (r *cacheArticleRepoStub) Update(article *model.Article) error {
	r.updateCount++
	copied := *article
	r.article = &copied
	return nil
}
func (r *cacheArticleRepoStub) Delete(id int64) error                               { return nil }
func (r *cacheArticleRepoStub) UpdateSlug(id int64, slug string) error              { return nil }
func (r *cacheArticleRepoStub) UpdateAIIndexStatus(id int64, status string) error   { return nil }
func (r *cacheArticleRepoStub) IncrementViewCount(id int64) error                   { return nil }
func (r *cacheArticleRepoStub) IncrementCommentCount(id int64) error                { return nil }
func (r *cacheArticleRepoStub) DecrementCommentCount(id int64) error                { return nil }
func (r *cacheArticleRepoStub) IncrementLikeCount(id int64) error                   { return nil }
func (r *cacheArticleRepoStub) DecrementLikeCount(id int64) error                   { return nil }
func (r *cacheArticleRepoStub) UpdateTop(id int64, isTop bool) error                { return nil }
func (r *cacheArticleRepoStub) UpdatePopularity(id int64, popularity float64) error { return nil }
func (r *cacheArticleRepoStub) GetAllPublished() ([]*model.Article, error)          { return nil, nil }
func (r *cacheArticleRepoStub) GetDueScheduledArticles() ([]*model.Article, error)  { return nil, nil }
func (r *cacheArticleRepoStub) PublishScheduled(articleID int64) error              { return nil }
func (r *cacheArticleRepoStub) AddInteraction(userID, articleID int64, interactionType string) (bool, error) {
	return false, nil
}
func (r *cacheArticleRepoStub) RemoveInteraction(userID, articleID int64, interactionType string) (bool, error) {
	return false, nil
}
func (r *cacheArticleRepoStub) GetInteractionState(userID, articleID int64) (*model.ArticleInteractionState, error) {
	return &model.ArticleInteractionState{}, nil
}
func (r *cacheArticleRepoStub) ListByInteraction(userID int64, interactionType string, filter repository.ArticleFilter) ([]*model.Article, int64, error) {
	return nil, 0, nil
}

type cacheCategoryRepoStub struct{}

func (r *cacheCategoryRepoStub) Create(category *model.Category) error { return nil }
func (r *cacheCategoryRepoStub) GetByID(id int64) (*model.Category, error) {
	return &model.Category{ID: id, Name: "Tech", Slug: "tech"}, nil
}
func (r *cacheCategoryRepoStub) GetBySlug(slug string) (*model.Category, error) { return nil, nil }
func (r *cacheCategoryRepoStub) List() ([]*model.Category, error)               { return nil, nil }
func (r *cacheCategoryRepoStub) ListPaginated(filter repository.CategoryFilter) ([]*model.Category, int64, error) {
	return nil, 0, nil
}
func (r *cacheCategoryRepoStub) Update(category *model.Category) error { return nil }
func (r *cacheCategoryRepoStub) Delete(id int64) error                 { return nil }
func (r *cacheCategoryRepoStub) IncrementArticleCount(id int64) error  { return nil }
func (r *cacheCategoryRepoStub) DecrementArticleCount(id int64) error  { return nil }

func TestArticleServiceGetBySlug_UsesCachedDetail(t *testing.T) {
	repo := &cacheArticleRepoStub{
		article: &model.Article{ID: 7, Title: "cached", Slug: "cached", Status: "published"},
	}
	cache := newMemoryArticleCacheStore()
	payload, err := json.Marshal(repo.article)
	if err != nil {
		t.Fatalf("failed to marshal article: %v", err)
	}
	if err := cache.Set(context.Background(), articleDetailSlugKey("cached"), payload, time.Minute); err != nil {
		t.Fatalf("failed to seed cache: %v", err)
	}
	svc := newArticleServiceWithCacheStore(repo, &cacheCategoryRepoStub{}, cache, nil, nil)

	article, err := svc.GetBySlug("cached")
	if err != nil {
		t.Fatalf("expected cached lookup success, got %v", err)
	}
	if repo.getBySlugCount != 0 {
		t.Fatalf("expected cache hit to avoid repository lookup, got %d lookups", repo.getBySlugCount)
	}
	if article == nil || article.ID != 7 || article.Slug != "cached" {
		t.Fatalf("unexpected cached article: %#v", article)
	}
}

func TestArticleServiceList_UsesCachedResultsUntilInvalidated(t *testing.T) {
	repo := &cacheArticleRepoStub{
		article: &model.Article{ID: 7, Title: "first", Slug: "first", Status: "published"},
	}
	cache := newMemoryArticleCacheStore()
	svc := newArticleServiceWithCacheStore(repo, &cacheCategoryRepoStub{}, cache, nil, nil)

	first, total, err := svc.List("published", 3, "go", true, 2, 9)
	if err != nil {
		t.Fatalf("expected first list success, got %v", err)
	}
	if repo.listCount != 1 {
		t.Fatalf("expected one repository lookup, got %d", repo.listCount)
	}
	second, totalAgain, err := svc.List("published", 3, "go", true, 2, 9)
	if err != nil {
		t.Fatalf("expected cached list success, got %v", err)
	}
	if repo.listCount != 1 {
		t.Fatalf("expected second call to hit cache, got %d repository lookups", repo.listCount)
	}
	if total != totalAgain || len(first) != len(second) || first[0].ID != second[0].ID {
		t.Fatalf("expected cached result to match original response")
	}

	repo.article.Title = "updated"
	if _, err := svc.Update(7, "updated", "content enough for update cache test", "summary", 3, nil); err != nil {
		t.Fatalf("expected update success, got %v", err)
	}

	afterUpdate, _, err := svc.List("published", 3, "go", true, 2, 9)
	if err != nil {
		t.Fatalf("expected list after invalidation to succeed, got %v", err)
	}
	if repo.listCount != 2 {
		t.Fatalf("expected collection cache invalidation to force a repo lookup, got %d", repo.listCount)
	}
	if len(afterUpdate) == 0 || afterUpdate[0].Title != "updated" {
		t.Fatalf("expected updated content after cache invalidation, got %#v", afterUpdate)
	}
}

func TestArticleServiceListOrbitArticles_UsesCache(t *testing.T) {
	repo := &cacheArticleRepoStub{
		article: &model.Article{ID: 8, Title: "orbit", Slug: "orbit", Status: "published"},
	}
	cache := newMemoryArticleCacheStore()
	svc := newArticleServiceWithCacheStore(repo, &cacheCategoryRepoStub{}, cache, nil, nil)

	if _, err := svc.ListOrbitArticles(); err != nil {
		t.Fatalf("expected first orbit lookup success, got %v", err)
	}
	if repo.orbitCount != 1 {
		t.Fatalf("expected one orbit repository lookup, got %d", repo.orbitCount)
	}
	if _, err := svc.ListOrbitArticles(); err != nil {
		t.Fatalf("expected cached orbit lookup success, got %v", err)
	}
	if repo.orbitCount != 1 {
		t.Fatalf("expected orbit cache hit to avoid repo lookup, got %d", repo.orbitCount)
	}
}
