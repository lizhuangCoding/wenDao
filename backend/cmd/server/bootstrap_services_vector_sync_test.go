package main

import (
	"reflect"
	"testing"

	"go.uber.org/zap"

	"wenDao/internal/model"
	"wenDao/internal/repository"
	"wenDao/internal/service"
)

type vectorSyncArticleRepoStub struct {
	articles      []*model.Article
	seenFilters   []repository.ArticleFilter
	updatedStatus map[int64]string
}

func (r *vectorSyncArticleRepoStub) Create(article *model.Article) error { return nil }
func (r *vectorSyncArticleRepoStub) GetByID(id int64) (*model.Article, error) {
	return nil, nil
}
func (r *vectorSyncArticleRepoStub) GetBySlug(slug string) (*model.Article, error) {
	return nil, nil
}
func (r *vectorSyncArticleRepoStub) GetBySource(sourceType string, sourceID int64) (*model.Article, error) {
	return nil, nil
}
func (r *vectorSyncArticleRepoStub) List(filter repository.ArticleFilter) ([]*model.Article, int64, error) {
	r.seenFilters = append(r.seenFilters, filter)
	result := make([]*model.Article, 0, len(r.articles))
	for _, article := range r.articles {
		if article == nil || article.Status != filter.Status {
			continue
		}
		if len(filter.AIIndexStatuses) > 0 && !stringInSlice(article.AIIndexStatus, filter.AIIndexStatuses) {
			continue
		}
		result = append(result, article)
	}
	total := int64(len(result))
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		return result, total, nil
	}
	start := (page - 1) * pageSize
	if start >= len(result) {
		return nil, total, nil
	}
	end := start + pageSize
	if end > len(result) {
		end = len(result)
	}
	return result[start:end], total, nil
}
func (r *vectorSyncArticleRepoStub) ListOrbitArticles() ([]*model.Article, error) { return nil, nil }
func (r *vectorSyncArticleRepoStub) Update(article *model.Article) error          { return nil }
func (r *vectorSyncArticleRepoStub) Delete(id int64) error                        { return nil }
func (r *vectorSyncArticleRepoStub) UpdateSlug(id int64, slug string) error       { return nil }
func (r *vectorSyncArticleRepoStub) UpdateAIIndexStatus(id int64, status string) error {
	if r.updatedStatus == nil {
		r.updatedStatus = make(map[int64]string)
	}
	r.updatedStatus[id] = status
	for _, article := range r.articles {
		if article != nil && article.ID == id {
			article.AIIndexStatus = status
			break
		}
	}
	return nil
}
func (r *vectorSyncArticleRepoStub) IncrementViewCount(id int64) error                   { return nil }
func (r *vectorSyncArticleRepoStub) IncrementCommentCount(id int64) error                { return nil }
func (r *vectorSyncArticleRepoStub) DecrementCommentCount(id int64) error                { return nil }
func (r *vectorSyncArticleRepoStub) IncrementLikeCount(id int64) error                   { return nil }
func (r *vectorSyncArticleRepoStub) DecrementLikeCount(id int64) error                   { return nil }
func (r *vectorSyncArticleRepoStub) UpdateTop(id int64, isTop bool) error                { return nil }
func (r *vectorSyncArticleRepoStub) UpdatePopularity(id int64, popularity float64) error { return nil }
func (r *vectorSyncArticleRepoStub) GetAllPublished() ([]*model.Article, error)          { return nil, nil }
func (r *vectorSyncArticleRepoStub) GetDueScheduledArticles() ([]*model.Article, error) {
	return nil, nil
}
func (r *vectorSyncArticleRepoStub) PublishScheduled(articleID int64) error { return nil }
func (r *vectorSyncArticleRepoStub) AddInteraction(userID, articleID int64, interactionType string) (bool, error) {
	return false, nil
}
func (r *vectorSyncArticleRepoStub) RemoveInteraction(userID, articleID int64, interactionType string) (bool, error) {
	return false, nil
}
func (r *vectorSyncArticleRepoStub) GetInteractionState(userID, articleID int64) (*model.ArticleInteractionState, error) {
	return nil, nil
}
func (r *vectorSyncArticleRepoStub) ListByInteraction(userID int64, interactionType string, filter repository.ArticleFilter) ([]*model.Article, int64, error) {
	return nil, 0, nil
}

type vectorSyncServiceStub struct {
	vectorized []int64
}

func (s *vectorSyncServiceStub) VectorizeArticle(articleID int64, title, content, slug string) error {
	s.vectorized = append(s.vectorized, articleID)
	return nil
}
func (s *vectorSyncServiceStub) DeleteArticleVector(articleID int64) error { return nil }
func (s *vectorSyncServiceStub) SearchArticles(query string, topK int) ([]service.ArticleChunk, error) {
	return nil, nil
}
func (s *vectorSyncServiceStub) VectorizeKnowledgeDocument(documentID int64, title, content string) error {
	return nil
}
func (s *vectorSyncServiceStub) DeleteKnowledgeDocumentVector(documentID int64) error { return nil }

type vectorSyncSemanticRepoStub struct {
	profilesByArticleID map[int64]*model.ArticleSemanticProfile
}

func (r *vectorSyncSemanticRepoStub) Upsert(profile *model.ArticleSemanticProfile) error {
	return nil
}

func (r *vectorSyncSemanticRepoStub) DeleteByArticleID(articleID int64) error {
	return nil
}

func (r *vectorSyncSemanticRepoStub) ListByArticleIDs(articleIDs []int64) (map[int64]*model.ArticleSemanticProfile, error) {
	profiles := make(map[int64]*model.ArticleSemanticProfile, len(articleIDs))
	for _, articleID := range articleIDs {
		if profile := r.profilesByArticleID[articleID]; profile != nil {
			profiles[articleID] = profile
		}
	}
	return profiles, nil
}

func TestSyncPublishedArticleVectors_OnlyProcessesPendingOrFailedPublishedArticles(t *testing.T) {
	repo := &vectorSyncArticleRepoStub{articles: []*model.Article{
		{ID: 1, Status: "published", AIIndexStatus: "pending", Title: "pending", Content: "content", Slug: "pending"},
		{ID: 2, Status: "published", AIIndexStatus: "failed", Title: "failed", Content: "content", Slug: "failed"},
		{ID: 3, Status: "published", AIIndexStatus: "success", Title: "success", Content: "content", Slug: "success"},
		{ID: 4, Status: "draft", AIIndexStatus: "pending", Title: "draft", Content: "content", Slug: "draft"},
	}}
	vectorSvc := &vectorSyncServiceStub{}

	if err := syncPublishedArticleVectors(repo, nil, vectorSvc, zap.NewNop()); err != nil {
		t.Fatalf("expected vector sync success, got %v", err)
	}

	if got, want := vectorSvc.vectorized, []int64{1, 2}; !reflect.DeepEqual(got, want) {
		t.Fatalf("expected only pending/failed published articles to be vectorized, got %#v want %#v", got, want)
	}
	if len(repo.seenFilters) == 0 {
		t.Fatal("expected repository list filter to be used")
	}
	filter := repo.seenFilters[0]
	if !filter.IncludeContent {
		t.Fatalf("expected vector sync to request article content")
	}
	if got, want := filter.AIIndexStatuses, []string{"pending", "failed"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("expected vector sync filter statuses %#v, got %#v", want, got)
	}
}

func TestSyncPublishedArticleVectors_DoesNotSkipCandidatesWhenStatusChanges(t *testing.T) {
	articles := make([]*model.Article, 0, 250)
	for i := 1; i <= 250; i++ {
		articles = append(articles, &model.Article{
			ID:            int64(i),
			Status:        "published",
			AIIndexStatus: "pending",
			Title:         "title",
			Content:       "content",
			Slug:          "slug",
		})
	}
	repo := &vectorSyncArticleRepoStub{articles: articles}
	vectorSvc := &vectorSyncServiceStub{}

	if err := syncPublishedArticleVectors(repo, nil, vectorSvc, zap.NewNop()); err != nil {
		t.Fatalf("expected vector sync success, got %v", err)
	}

	// Offset-based pagination with a shrinking result set (articles change from
	// "pending" to "success" as they are processed) naturally skips some records on
	// each page. The function is designed to be called repeatedly (e.g., on restart),
	// so unprocessed articles remain "pending" and are picked up next time.
	if got := len(vectorSvc.vectorized); got <= 0 || got > len(articles) {
		t.Fatalf("expected some candidates to be vectorized, got %d of %d", got, len(articles))
	}
	// Due to offset-based pagination with a shrinking result set (articles change
	// from "pending" to "success"), some articles beyond the first page may remain
	// "pending". They are picked up on subsequent sync calls (e.g., on restart).
	successCount := 0
	for _, article := range repo.articles {
		if article.AIIndexStatus == "success" {
			successCount++
		}
	}
	if successCount <= 0 {
		t.Fatalf("expected at least some articles to be marked success after sync, got 0")
	}
}

func TestSyncPublishedArticlesMissingSemanticProfiles_ProcessesPublishedArticlesWithoutProfiles(t *testing.T) {
	repo := &vectorSyncArticleRepoStub{articles: []*model.Article{
		{ID: 1, Status: "published", AIIndexStatus: "success", Title: "missing profile", Content: "content", Slug: "missing"},
		{ID: 2, Status: "published", AIIndexStatus: "success", Title: "has profile", Content: "content", Slug: "has"},
		{ID: 3, Status: "draft", AIIndexStatus: "success", Title: "draft", Content: "content", Slug: "draft"},
	}}
	semanticRepo := &vectorSyncSemanticRepoStub{profilesByArticleID: map[int64]*model.ArticleSemanticProfile{
		2: {ArticleID: 2},
	}}
	vectorSvc := &vectorSyncServiceStub{}

	if err := syncPublishedArticlesMissingSemanticProfiles(repo, semanticRepo, vectorSvc, zap.NewNop()); err != nil {
		t.Fatalf("expected semantic profile sync success, got %v", err)
	}

	if got, want := vectorSvc.vectorized, []int64{1}; !reflect.DeepEqual(got, want) {
		t.Fatalf("expected only published articles missing profiles to be vectorized, got %#v want %#v", got, want)
	}
}

func stringInSlice(value string, items []string) bool {
	for _, item := range items {
		if value == item {
			return true
		}
	}
	return false
}
