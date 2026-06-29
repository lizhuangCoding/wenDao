package asyncjob

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

	"wenDao/internal/model"
	asyncjobrepo "wenDao/internal/repository/asyncjob"
)

const (
	JobTypeNotificationCreate       = "notification_create"
	JobTypeCommentReplyEmail        = "comment_reply_email"
	JobTypeArticleCacheInvalidation = "article_cache_invalidation"
	JobTypeArticleVectorize         = "article_vectorize"
	JobTypeArticleVectorDelete      = "article_vector_delete"
)

type NotificationCreatePayload struct {
	UserID           int64  `json:"user_id"`
	NotificationType string `json:"notification_type"`
	Title            string `json:"title"`
	Content          string `json:"content"`
	LinkURL          string `json:"link_url"`
}

type CommentReplyEmailPayload struct {
	RecipientEmail      string `json:"recipient_email"`
	RecipientUsername   string `json:"recipient_username"`
	ReplyAuthorUsername string `json:"reply_author_username"`
	ArticleTitle        string `json:"article_title"`
	ArticleSlug         string `json:"article_slug"`
	CommentPreview      string `json:"comment_preview"`
}

type ArticleCacheInvalidationPayload struct {
	ArticleID              int64  `json:"article_id"`
	ArticleSlug            string `json:"article_slug"`
	BumpCollectionVersions bool   `json:"bump_collection_versions"`
}

type ArticleVectorizePayload struct {
	ArticleID int64  `json:"article_id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Slug      string `json:"slug"`
}

type ArticleVectorDeletePayload struct {
	ArticleID int64 `json:"article_id"`
}

type JobHandler func(ctx context.Context, job *model.AsyncJob) error

type Service interface {
	ProcessPending(ctx context.Context, limit int) error
}

type service struct {
	repo     asyncjobrepo.AsyncJobRepository
	logger   *zap.Logger
	clock    func() time.Time
	handlers map[string]JobHandler
}

type Option func(*service)

func WithClock(clock func() time.Time) Option {
	return func(s *service) {
		if clock != nil {
			s.clock = clock
		}
	}
}

func WithJobHandler(jobType string, handler JobHandler) Option {
	return func(s *service) {
		if jobType == "" || handler == nil {
			return
		}
		s.handlers[jobType] = handler
	}
}

func NewService(repo asyncjobrepo.AsyncJobRepository, logger *zap.Logger, options ...Option) Service {
	svc := &service{
		repo:     repo,
		logger:   logger,
		clock:    time.Now,
		handlers: make(map[string]JobHandler),
	}
	for _, option := range options {
		if option != nil {
			option(svc)
		}
	}
	return svc
}

func NewJob(jobType string, payload any) (*model.AsyncJob, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal async job payload: %w", err)
	}

	return &model.AsyncJob{
		JobType:     jobType,
		Status:      model.AsyncJobStatusPending,
		Payload:     body,
		Attempts:    0,
		MaxAttempts: 3,
		RunAfter:    time.Now(),
	}, nil
}

func (s *service) ProcessPending(ctx context.Context, limit int) error {
	if s == nil || s.repo == nil {
		return nil
	}

	now := s.clock()
	jobs, err := s.repo.ListRunnable(now, limit)
	if err != nil {
		return fmt.Errorf("list runnable async jobs: %w", err)
	}

	for _, job := range jobs {
		if job == nil {
			continue
		}
		claimed, claimErr := s.repo.Claim(job.ID, now)
		if claimErr != nil {
			return fmt.Errorf("claim async job %d: %w", job.ID, claimErr)
		}
		if !claimed {
			continue
		}

		handler := s.handlers[job.JobType]
		if handler == nil {
			s.failJob(job, now, fmt.Errorf("no handler registered for job type %s", job.JobType))
			continue
		}

		if err := handler(ctx, job); err != nil {
			s.failJob(job, now, err)
			continue
		}

		if err := s.repo.MarkSucceeded(job.ID, now); err != nil {
			return fmt.Errorf("mark async job %d succeeded: %w", job.ID, err)
		}
	}

	return nil
}

func (s *service) failJob(job *model.AsyncJob, now time.Time, err error) {
	if s == nil || s.repo == nil || job == nil {
		return
	}

	nextAttempt := job.Attempts + 1
	delay := time.Duration(nextAttempt*nextAttempt) * time.Minute
	runAfter := now.Add(delay)
	lastError := err.Error()
	if repoErr := s.repo.MarkFailed(job.ID, runAfter, lastError); repoErr != nil && s.logger != nil {
		s.logger.Error("Failed to mark async job failed", zap.Int64("job_id", job.ID), zap.Error(repoErr))
	}
	if s.logger != nil {
		s.logger.Warn("Async job execution failed",
			zap.Int64("job_id", job.ID),
			zap.String("job_type", job.JobType),
			zap.Error(err),
			zap.Time("run_after", runAfter))
	}
}
