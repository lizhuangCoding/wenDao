package article

import (
	"testing"
	"time"

	"go.uber.org/zap"

	"wenDao/internal/model"
	"wenDao/internal/repository"
)

type deleteBatchArticleRepoStub struct {
	deletedIDs      []int64
	deleteCallCount int
	articles        []*model.Article
}

func (r *deleteBatchArticleRepoStub) Create(article *model.Article) error { return nil }
func (r *deleteBatchArticleRepoStub) GetByID(id int64) (*model.Article, error) {
	r.deleteCallCount++
	return nil, nil
}
func (r *deleteBatchArticleRepoStub) GetBySlug(slug string) (*model.Article, error) {
	return nil, nil
}
func (r *deleteBatchArticleRepoStub) GetBySource(sourceType string, sourceID int64) (*model.Article, error) {
	return nil, nil
}
func (r *deleteBatchArticleRepoStub) List(filter repository.ArticleFilter) ([]*model.Article, int64, error) {
	return nil, 0, nil
}
func (r *deleteBatchArticleRepoStub) Search(filter repository.ArticleSearchFilter) ([]repository.ArticleSearchResult, int64, error) {
	return nil, 0, nil
}
func (r *deleteBatchArticleRepoStub) ListOrbitArticles() ([]*model.Article, error) { return nil, nil }
func (r *deleteBatchArticleRepoStub) Update(article *model.Article) error          { return nil }
func (r *deleteBatchArticleRepoStub) Delete(id int64) error {
	r.deleteCallCount++
	return nil
}
func (r *deleteBatchArticleRepoStub) DeleteBatch(ids []int64) ([]*model.Article, error) {
	r.deletedIDs = append([]int64(nil), ids...)
	return r.articles, nil
}
func (r *deleteBatchArticleRepoStub) UpdateSlug(id int64, slug string) error            { return nil }
func (r *deleteBatchArticleRepoStub) UpdateAIIndexStatus(id int64, status string) error { return nil }
func (r *deleteBatchArticleRepoStub) IncrementViewCount(id int64) error                 { return nil }
func (r *deleteBatchArticleRepoStub) IncrementCommentCount(id int64) error              { return nil }
func (r *deleteBatchArticleRepoStub) DecrementCommentCount(id int64) error              { return nil }
func (r *deleteBatchArticleRepoStub) IncrementLikeCount(id int64) error                 { return nil }
func (r *deleteBatchArticleRepoStub) DecrementLikeCount(id int64) error                 { return nil }
func (r *deleteBatchArticleRepoStub) UpdateTop(id int64, isTop bool) error              { return nil }
func (r *deleteBatchArticleRepoStub) UpdatePopularity(id int64, popularity float64) error {
	return nil
}
func (r *deleteBatchArticleRepoStub) UpdatePopularityScores(now time.Time) error { return nil }
func (r *deleteBatchArticleRepoStub) GetAllPublished() ([]*model.Article, error) { return nil, nil }
func (r *deleteBatchArticleRepoStub) GetDueScheduledArticles() ([]*model.Article, error) {
	return nil, nil
}
func (r *deleteBatchArticleRepoStub) PublishScheduled(articleID int64) error { return nil }
func (r *deleteBatchArticleRepoStub) AddInteraction(userID, articleID int64, interactionType string) (bool, error) {
	return false, nil
}
func (r *deleteBatchArticleRepoStub) RemoveInteraction(userID, articleID int64, interactionType string) (bool, error) {
	return false, nil
}
func (r *deleteBatchArticleRepoStub) GetInteractionState(userID, articleID int64) (*model.ArticleInteractionState, error) {
	return nil, nil
}
func (r *deleteBatchArticleRepoStub) ListByInteraction(userID int64, interactionType string, filter repository.ArticleFilter) ([]*model.Article, int64, error) {
	return nil, 0, nil
}

func TestArticleServiceDeleteBatchUsesRepositoryBatchDelete(t *testing.T) {
	repo := &deleteBatchArticleRepoStub{
		articles: []*model.Article{
			{ID: 3, Slug: "a", Status: "published"},
			{ID: 5, Slug: "b", Status: "draft"},
		},
	}
	cache := newMemoryArticleCacheStore()
	svc := newArticleServiceWithCacheStore(repo, &viewCountCategoryRepoStub{}, cache, nil, zap.NewNop())

	if err := svc.DeleteBatch([]int64{3, 5, 3}); err != nil {
		t.Fatalf("expected batch delete to succeed, got %v", err)
	}

	if repo.deleteCallCount != 0 {
		t.Fatalf("expected service not to call single article delete/get, got %d calls", repo.deleteCallCount)
	}
	if len(repo.deletedIDs) != 2 || repo.deletedIDs[0] != 3 || repo.deletedIDs[1] != 5 {
		t.Fatalf("expected unique ids [3 5], got %#v", repo.deletedIDs)
	}
}
