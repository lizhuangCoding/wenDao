package comment

import (
	"testing"
	"time"

	"wenDao/internal/model"
	"wenDao/internal/repository"
	asyncjobrepo "wenDao/internal/repository/asyncjob"
	asyncjobsvc "wenDao/internal/service/asyncjob"
)

type commentAsyncJobRepoStub struct {
	jobs []*model.AsyncJob
}

func (r *commentAsyncJobRepoStub) Enqueue(job *model.AsyncJob) error {
	copied := *job
	r.jobs = append(r.jobs, &copied)
	return nil
}

func (r *commentAsyncJobRepoStub) ListRunnable(now time.Time, limit int) ([]*model.AsyncJob, error) {
	return nil, nil
}

func (r *commentAsyncJobRepoStub) Claim(id int64, now time.Time) (bool, error) {
	return false, nil
}

func (r *commentAsyncJobRepoStub) MarkSucceeded(id int64, finishedAt time.Time) error {
	return nil
}

func (r *commentAsyncJobRepoStub) MarkFailed(id int64, runAfter time.Time, lastError string) error {
	return nil
}

var _ asyncjobrepo.AsyncJobRepository = (*commentAsyncJobRepoStub)(nil)

type commentTxRunnerStub struct {
	jobRepo     *commentAsyncJobRepoStub
	commentRepo repository.CommentRepository
	articleRepo repository.ArticleRepository
	committed   bool
	rollbacked  bool
}

func (r *commentTxRunnerStub) Run(fn func(repository.CommentRepository, repository.ArticleRepository, asyncjobrepo.AsyncJobRepository) error) error {
	if err := fn(r.commentRepo, r.articleRepo, r.jobRepo); err != nil {
		r.rollbacked = true
		return err
	}
	r.committed = true
	return nil
}

type countFailingArticleRepo struct {
	replyNotificationArticleRepo
	incrementErr error
}

func (r *countFailingArticleRepo) IncrementCommentCount(id int64) error {
	return r.incrementErr
}

func TestCommentServiceCreateEnqueuesAsyncJobsInsideTransaction(t *testing.T) {
	recipient := &model.User{
		ID:                       12,
		Username:                 "reader",
		Email:                    "reader@example.com",
		CommentReplyEmailEnabled: true,
		Status:                   "active",
	}
	replyAuthor := &model.User{ID: 34, Username: "author", Email: "author@example.com", Status: "active"}
	parentID := int64(56)
	article := &model.Article{ID: 7, Title: "一篇文章", Slug: "essay", Status: "published"}
	commentRepo := &replyNotificationCommentRepo{
		parent: &model.Comment{
			ID:        parentID,
			ArticleID: article.ID,
			UserID:    recipient.ID,
			User:      recipient,
			Status:    "normal",
		},
		replyAuthor: replyAuthor,
		recipient:   recipient,
	}
	jobRepo := &commentAsyncJobRepoStub{}
	runner := &commentTxRunnerStub{
		jobRepo:     jobRepo,
		commentRepo: commentRepo,
		articleRepo: &replyNotificationArticleRepo{article: article},
	}
	svc := NewCommentService(
		commentRepo,
		&replyNotificationArticleRepo{article: article},
		WithWriteTransactionRunner(runner),
	)

	if _, err := svc.Create(article.ID, replyAuthor.ID, "谢谢你的评论。", &parentID, nil); err != nil {
		t.Fatalf("expected create to succeed, got %v", err)
	}

	if !runner.committed || runner.rollbacked {
		t.Fatalf("expected transaction commit, got committed=%v rollbacked=%v", runner.committed, runner.rollbacked)
	}
	if len(jobRepo.jobs) != 3 {
		t.Fatalf("expected 3 async jobs (cache + in-app + email), got %d", len(jobRepo.jobs))
	}

	jobTypes := []string{jobRepo.jobs[0].JobType, jobRepo.jobs[1].JobType, jobRepo.jobs[2].JobType}
	expected := map[string]bool{
		asyncjobsvc.JobTypeArticleCacheInvalidation: true,
		asyncjobsvc.JobTypeNotificationCreate:       true,
		asyncjobsvc.JobTypeCommentReplyEmail:        true,
	}
	for _, jobType := range jobTypes {
		if !expected[jobType] {
			t.Fatalf("unexpected job types: %#v", jobTypes)
		}
	}
}

func TestCommentServiceCreateRollsBackWhenIncrementCommentCountFails(t *testing.T) {
	parentID := int64(56)
	article := &model.Article{ID: 7, Title: "一篇文章", Slug: "essay", Status: "published"}
	commentRepo := &replyNotificationCommentRepo{
		parent: &model.Comment{
			ID:        parentID,
			ArticleID: article.ID,
			UserID:    12,
			Status:    "normal",
		},
		replyAuthor: &model.User{ID: 34, Username: "author", Status: "active"},
		recipient:   &model.User{ID: 12, Username: "reader", Status: "active"},
	}
	jobRepo := &commentAsyncJobRepoStub{}
	runner := &commentTxRunnerStub{
		jobRepo:     jobRepo,
		commentRepo: commentRepo,
		articleRepo: &countFailingArticleRepo{
			replyNotificationArticleRepo: replyNotificationArticleRepo{article: article},
			incrementErr:                 errCommentCountFailed,
		},
	}
	svc := NewCommentService(
		commentRepo,
		&replyNotificationArticleRepo{article: article},
		WithWriteTransactionRunner(runner),
	)

	if _, err := svc.Create(article.ID, 34, "谢谢你的评论。", &parentID, nil); err == nil {
		t.Fatal("expected create to fail when comment count update fails")
	}

	if !runner.rollbacked || runner.committed {
		t.Fatalf("expected rollback without commit, got committed=%v rollbacked=%v", runner.committed, runner.rollbacked)
	}
	if len(jobRepo.jobs) != 0 {
		t.Fatalf("expected no async jobs to be committed on rollback, got %d", len(jobRepo.jobs))
	}
}

func TestCommentServiceLikeEnqueuesReactionNotificationJob(t *testing.T) {
	author := &model.User{ID: 88, Username: "writer", Status: "active"}
	article := &model.Article{ID: 7, Title: "评论文章", Slug: "comment-article", Status: "published"}
	commentRepo := &likeNotificationCommentRepo{
		comment: &model.Comment{
			ID:        42,
			ArticleID: article.ID,
			UserID:    author.ID,
			User:      author,
			Content:   "这是一条会收到点赞的评论",
			Status:    "normal",
		},
	}
	jobRepo := &commentAsyncJobRepoStub{}
	runner := &commentTxRunnerStub{
		jobRepo:     jobRepo,
		commentRepo: commentRepo,
		articleRepo: &replyNotificationArticleRepo{article: article},
	}
	notifSvc := &recordingNotificationService{}
	userRepo := &actorUserRepo{users: map[int64]*model.User{7: {ID: 7, Username: "alice"}}}
	svc := NewCommentService(
		commentRepo,
		&replyNotificationArticleRepo{article: article},
		WithWriteTransactionRunner(runner),
		WithNotificationService(notifSvc),
		WithUserRepository(userRepo),
	)

	if err := svc.Like(42, 7); err != nil {
		t.Fatalf("expected like to succeed, got %v", err)
	}
	if len(notifSvc.creates) != 0 {
		t.Fatalf("expected reaction notification to enqueue async job instead of direct send, got %d direct notifications", len(notifSvc.creates))
	}
	if len(jobRepo.jobs) != 1 || jobRepo.jobs[0].JobType != asyncjobsvc.JobTypeNotificationCreate {
		t.Fatalf("expected one notification_create job, got %#v", jobRepo.jobs)
	}
}

var errCommentCountFailed = repositoryErr("comment count failed")

type repositoryErr string

func (e repositoryErr) Error() string { return string(e) }
