package article

import (
	"testing"
	"time"

	"go.uber.org/zap"

	"wenDao/internal/model"
	"wenDao/internal/repository"
	asyncjobrepo "wenDao/internal/repository/asyncjob"
	aisvc "wenDao/internal/service/ai"
	asyncjobsvc "wenDao/internal/service/asyncjob"
)

type articleAsyncJobRepoStub struct {
	jobs []*model.AsyncJob
}

func (r *articleAsyncJobRepoStub) Enqueue(job *model.AsyncJob) error {
	copied := *job
	r.jobs = append(r.jobs, &copied)
	return nil
}

func (r *articleAsyncJobRepoStub) ListRunnable(now time.Time, limit int) ([]*model.AsyncJob, error) {
	return nil, nil
}

func (r *articleAsyncJobRepoStub) Claim(id int64, now time.Time) (bool, error) {
	return false, nil
}

func (r *articleAsyncJobRepoStub) MarkSucceeded(id int64, finishedAt time.Time) error {
	return nil
}

func (r *articleAsyncJobRepoStub) MarkFailed(id int64, runAfter time.Time, lastError string) error {
	return nil
}

var _ asyncjobrepo.AsyncJobRepository = (*articleAsyncJobRepoStub)(nil)

type articleTxRunnerStub struct {
	jobRepo      *articleAsyncJobRepoStub
	articleRepo  repository.ArticleRepository
	categoryRepo repository.CategoryRepository
}

func (r *articleTxRunnerStub) Run(fn func(repository.ArticleRepository, repository.CategoryRepository, asyncjobrepo.AsyncJobRepository) error) error {
	return fn(r.articleRepo, r.categoryRepo, r.jobRepo)
}

type vectorRecordingArticleRepo struct {
	cacheArticleRepoStub
	created []*model.Article
}

func (r *vectorRecordingArticleRepo) Create(article *model.Article) error {
	article.ID = 9
	copied := *article
	r.article = &copied
	r.created = append(r.created, &copied)
	return nil
}

func (r *vectorRecordingArticleRepo) UpdateSlug(id int64, slug string) error {
	if r.article != nil {
		r.article.Slug = slug
	}
	return nil
}

type vectorServiceStub struct{}

func (s *vectorServiceStub) VectorizeArticle(articleID int64, title, content, slug string) error {
	return nil
}

func (s *vectorServiceStub) DeleteArticleVector(articleID int64) error {
	return nil
}

func (s *vectorServiceStub) SearchArticles(query string, topK int) ([]aisvc.ArticleChunk, error) {
	return nil, nil
}

func (s *vectorServiceStub) VectorizeKnowledgeDocument(documentID int64, title, content string) error {
	return nil
}

func (s *vectorServiceStub) DeleteKnowledgeDocumentVector(documentID int64) error {
	return nil
}

func TestArticleServiceCreatePublishedArticleEnqueuesVectorizeJob(t *testing.T) {
	repo := &vectorRecordingArticleRepo{}
	jobRepo := &articleAsyncJobRepoStub{}
	svc := newArticleServiceWithCacheStore(
		repo,
		&cacheCategoryRepoStub{},
		newMemoryArticleCacheStore(),
		&vectorServiceStub{},
		zap.NewNop(),
		WithWriteTransactionRunner(&articleTxRunnerStub{
			jobRepo:      jobRepo,
			articleRepo:  repo,
			categoryRepo: &cacheCategoryRepoStub{},
		}),
	)

	article, err := svc.Create("已发布文章", "正文", "摘要", 3, 5, nil, "published")
	if err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}
	if article == nil || article.ID != 9 {
		t.Fatalf("expected created article with id 9, got %#v", article)
	}
	if len(jobRepo.jobs) != 1 || jobRepo.jobs[0].JobType != asyncjobsvc.JobTypeArticleVectorize {
		t.Fatalf("expected one vectorize job, got %#v", jobRepo.jobs)
	}
}

func TestArticleServiceDraftEnqueuesDeleteVectorJob(t *testing.T) {
	repo := &cacheArticleRepoStub{
		article: &model.Article{
			ID:         7,
			Title:      "published",
			Slug:       "published",
			Status:     "published",
			CategoryID: 3,
		},
	}
	jobRepo := &articleAsyncJobRepoStub{}
	svc := newArticleServiceWithCacheStore(
		repo,
		&cacheCategoryRepoStub{},
		newMemoryArticleCacheStore(),
		&vectorServiceStub{},
		zap.NewNop(),
		WithWriteTransactionRunner(&articleTxRunnerStub{
			jobRepo:      jobRepo,
			articleRepo:  repo,
			categoryRepo: &cacheCategoryRepoStub{},
		}),
	)

	if err := svc.Draft(7); err != nil {
		t.Fatalf("expected draft transition to succeed, got %v", err)
	}
	if len(jobRepo.jobs) != 1 || jobRepo.jobs[0].JobType != asyncjobsvc.JobTypeArticleVectorDelete {
		t.Fatalf("expected one vector delete job, got %#v", jobRepo.jobs)
	}
}
