package article

import (
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"wenDao/internal/model"
	"wenDao/internal/repository"
)

type viewCountArticleRepoStub struct {
	articleByID   *model.Article
	articleBySlug *model.Article
	updated       *model.Article
	incrementCh   chan int64
}

func (r *viewCountArticleRepoStub) Create(article *model.Article) error { return nil }
func (r *viewCountArticleRepoStub) GetByID(id int64) (*model.Article, error) {
	return r.articleByID, nil
}
func (r *viewCountArticleRepoStub) GetBySlug(slug string) (*model.Article, error) {
	return r.articleBySlug, nil
}
func (r *viewCountArticleRepoStub) GetBySource(sourceType string, sourceID int64) (*model.Article, error) {
	return nil, nil
}
func (r *viewCountArticleRepoStub) List(filter repository.ArticleFilter) ([]*model.Article, int64, error) {
	return nil, 0, nil
}
func (r *viewCountArticleRepoStub) ListOrbitArticles() ([]*model.Article, error) { return nil, nil }
func (r *viewCountArticleRepoStub) Update(article *model.Article) error {
	copied := *article
	r.updated = &copied
	return nil
}
func (r *viewCountArticleRepoStub) Delete(id int64) error                               { return nil }
func (r *viewCountArticleRepoStub) UpdateSlug(id int64, slug string) error              { return nil }
func (r *viewCountArticleRepoStub) UpdateAIIndexStatus(id int64, status string) error   { return nil }
func (r *viewCountArticleRepoStub) IncrementCommentCount(id int64) error                { return nil }
func (r *viewCountArticleRepoStub) DecrementCommentCount(id int64) error                { return nil }
func (r *viewCountArticleRepoStub) IncrementLikeCount(id int64) error                   { return nil }
func (r *viewCountArticleRepoStub) DecrementLikeCount(id int64) error                   { return nil }
func (r *viewCountArticleRepoStub) UpdateTop(id int64, isTop bool) error                { return nil }
func (r *viewCountArticleRepoStub) UpdatePopularity(id int64, popularity float64) error { return nil }
func (r *viewCountArticleRepoStub) GetAllPublished() ([]*model.Article, error)          { return nil, nil }
func (r *viewCountArticleRepoStub) GetDueScheduledArticles() ([]*model.Article, error) {
	return nil, nil
}
func (r *viewCountArticleRepoStub) PublishScheduled(articleID int64) error { return nil }
func (r *viewCountArticleRepoStub) AddInteraction(userID, articleID int64, interactionType string) (bool, error) {
	return false, nil
}
func (r *viewCountArticleRepoStub) RemoveInteraction(userID, articleID int64, interactionType string) (bool, error) {
	return false, nil
}
func (r *viewCountArticleRepoStub) GetInteractionState(userID, articleID int64) (*model.ArticleInteractionState, error) {
	return &model.ArticleInteractionState{}, nil
}
func (r *viewCountArticleRepoStub) ListByInteraction(userID int64, interactionType string, filter repository.ArticleFilter) ([]*model.Article, int64, error) {
	return nil, 0, nil
}
func (r *viewCountArticleRepoStub) IncrementViewCount(id int64) error {
	if r.incrementCh != nil {
		r.incrementCh <- id
	}
	return nil
}

type viewCountCategoryRepoStub struct{}

func (r *viewCountCategoryRepoStub) Create(category *model.Category) error          { return nil }
func (r *viewCountCategoryRepoStub) GetByID(id int64) (*model.Category, error)      { return nil, nil }
func (r *viewCountCategoryRepoStub) GetBySlug(slug string) (*model.Category, error) { return nil, nil }
func (r *viewCountCategoryRepoStub) List() ([]*model.Category, error)               { return nil, nil }
func (r *viewCountCategoryRepoStub) ListPaginated(filter repository.CategoryFilter) ([]*model.Category, int64, error) {
	return nil, 0, nil
}
func (r *viewCountCategoryRepoStub) Update(category *model.Category) error { return nil }
func (r *viewCountCategoryRepoStub) Delete(id int64) error                 { return nil }
func (r *viewCountCategoryRepoStub) IncrementArticleCount(id int64) error  { return nil }
func (r *viewCountCategoryRepoStub) DecrementArticleCount(id int64) error  { return nil }

func newArticleServiceForViewCountTest(repo repository.ArticleRepository) ArticleService {
	return NewArticleService(
		repo,
		&viewCountCategoryRepoStub{},
		redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"}),
		nil,
		nil,
	)
}

func TestArticleServiceGetByID_DoesNotIncrementViewCount(t *testing.T) {
	repo := &viewCountArticleRepoStub{
		articleByID: &model.Article{ID: 12, Title: "draft", Status: "draft"},
		incrementCh: make(chan int64, 1),
	}
	svc := newArticleServiceForViewCountTest(repo)

	_, err := svc.GetByID(12)
	if err != nil {
		t.Fatalf("expected article lookup to succeed, got %v", err)
	}

	select {
	case id := <-repo.incrementCh:
		t.Fatalf("expected GetByID to avoid incrementing view count, got increment for article %d", id)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestArticleServiceGetBySlug_DoesNotIncrementViewCount(t *testing.T) {
	repo := &viewCountArticleRepoStub{
		articleBySlug: &model.Article{ID: 21, Title: "published", Status: "published", Slug: "published"},
		incrementCh:   make(chan int64, 1),
	}
	svc := newArticleServiceForViewCountTest(repo)

	_, err := svc.GetBySlug("published")
	if err != nil {
		t.Fatalf("expected article lookup to succeed, got %v", err)
	}

	select {
	case id := <-repo.incrementCh:
		t.Fatalf("expected GetBySlug to avoid incrementing view count, got increment for article %d", id)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestArticleServiceUpdatePublishedArticle_PersistsPendingAIIndexStatus(t *testing.T) {
	repo := &viewCountArticleRepoStub{
		articleByID: &model.Article{
			ID:            31,
			Title:         "旧标题",
			Content:       "旧正文",
			Summary:       "旧摘要",
			CategoryID:    7,
			Status:        "published",
			AIIndexStatus: "success",
			Slug:          "old",
		},
	}
	svc := newArticleServiceForViewCountTest(repo)

	_, err := svc.Update(31, "新标题", "新正文", "新摘要", 7, nil)
	if err != nil {
		t.Fatalf("expected update success, got %v", err)
	}
	if repo.updated == nil {
		t.Fatal("expected article update to be persisted")
	}
	if repo.updated.AIIndexStatus != "pending" {
		t.Fatalf("expected published content update to persist pending AI index status, got %q", repo.updated.AIIndexStatus)
	}
}

func TestArticleServicePublishDraft_PersistsPendingAIIndexStatusBeforeAsyncVectorization(t *testing.T) {
	repo := &viewCountArticleRepoStub{
		articleByID: &model.Article{
			ID:            32,
			Title:         "草稿",
			Content:       "正文",
			Summary:       "摘要",
			CategoryID:    7,
			Status:        "draft",
			AIIndexStatus: "success",
			Slug:          "draft",
		},
	}
	svc := newArticleServiceForViewCountTest(repo)

	if err := svc.Publish(32); err != nil {
		t.Fatalf("expected publish success, got %v", err)
	}
	if repo.updated == nil {
		t.Fatal("expected publish to persist article update")
	}
	if repo.updated.Status != "published" {
		t.Fatalf("expected article to be published, got %q", repo.updated.Status)
	}
	if repo.updated.AIIndexStatus != "pending" {
		t.Fatalf("expected publish to persist pending AI index status, got %q", repo.updated.AIIndexStatus)
	}
}
