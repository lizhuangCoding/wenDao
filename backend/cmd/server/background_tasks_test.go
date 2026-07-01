package main

import (
	"context"
	"mime/multipart"
	"testing"
	"time"

	"go.uber.org/zap"

	"wenDao/config"
	"wenDao/internal/model"
	"wenDao/internal/pkg/async"
	"wenDao/internal/service"
)

type immediateCleanupUploadService struct {
	called chan time.Time
}

func (s *immediateCleanupUploadService) UploadImage(file multipart.File, header *multipart.FileHeader, userID int64) (*model.Upload, error) {
	return nil, nil
}

func (s *immediateCleanupUploadService) UploadArticleImage(file multipart.File, header *multipart.FileHeader, userID int64) (*model.Upload, error) {
	return nil, nil
}

func (s *immediateCleanupUploadService) UploadCoverImage(file multipart.File, header *multipart.FileHeader, userID int64) (*model.Upload, error) {
	return nil, nil
}

func (s *immediateCleanupUploadService) CleanupByFilePath(filePath string) error {
	return nil
}

func (s *immediateCleanupUploadService) CleanupUnreferenced(now time.Time) (service.UploadCleanupResult, error) {
	s.called <- now
	return service.UploadCleanupResult{}, nil
}

func TestStartUploadCleanupScheduler_RunsCleanupImmediately(t *testing.T) {
	uploadService := &immediateCleanupUploadService{called: make(chan time.Time, 1)}
	runner := async.NewTaskRunner(context.Background(), zap.NewNop())
	task := uploadCleanupTask(&config.Config{
		Upload: config.UploadConfig{
			CleanupEnabled:       true,
			CleanupIntervalHours: 24,
		},
	}, uploadService)
	if !task.enabled {
		t.Fatal("expected upload cleanup task to be enabled")
	}
	if err := task.start(context.Background(), runner, zap.NewNop()); err != nil {
		t.Fatalf("expected upload cleanup task to start, got %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = runner.Shutdown(shutdownCtx)
	}()

	select {
	case <-uploadService.called:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("expected upload cleanup to run immediately after scheduler starts")
	}
}

type immediateArticleSchedulerService struct {
	published chan int64
}

func (s *immediateArticleSchedulerService) Create(title, content, summary string, categoryID, authorID int64, coverImage *string, status string) (*model.Article, error) {
	return nil, nil
}
func (s *immediateArticleSchedulerService) GetByID(id int64) (*model.Article, error) {
	return nil, nil
}
func (s *immediateArticleSchedulerService) GetBySlug(slug string) (*model.Article, error) {
	return nil, nil
}
func (s *immediateArticleSchedulerService) List(status string, categoryID, tagID int64, keyword string, sortByPopularity bool, page, pageSize int) ([]*model.Article, int64, error) {
	return nil, 0, nil
}
func (s *immediateArticleSchedulerService) SearchArticles(keyword string, categoryID, tagID int64, page, pageSize int) ([]service.ArticleSearchResult, int64, error) {
	return nil, 0, nil
}
func (s *immediateArticleSchedulerService) ListOrbitArticles() ([]*model.Article, error) {
	return nil, nil
}
func (s *immediateArticleSchedulerService) Update(id int64, title, content, summary string, categoryID int64, coverImage *string) (*model.Article, error) {
	return nil, nil
}
func (s *immediateArticleSchedulerService) SetTags(id int64, tagIDs []int64) (*model.Article, error) {
	return nil, nil
}
func (s *immediateArticleSchedulerService) Delete(id int64) error         { return nil }
func (s *immediateArticleSchedulerService) DeleteBatch(ids []int64) error { return nil }
func (s *immediateArticleSchedulerService) Publish(id int64) error        { return nil }
func (s *immediateArticleSchedulerService) Draft(id int64) error          { return nil }
func (s *immediateArticleSchedulerService) AutoSave(id int64, title, content, summary string) error {
	return nil
}
func (s *immediateArticleSchedulerService) IncrViewCount(id int64) error { return nil }
func (s *immediateArticleSchedulerService) LikeArticle(id int64) error   { return nil }
func (s *immediateArticleSchedulerService) UnlikeArticle(id int64) error { return nil }
func (s *immediateArticleSchedulerService) LikeArticleForUser(userID, articleID int64) (*model.ArticleInteractionState, error) {
	return nil, nil
}
func (s *immediateArticleSchedulerService) UnlikeArticleForUser(userID, articleID int64) (*model.ArticleInteractionState, error) {
	return nil, nil
}
func (s *immediateArticleSchedulerService) FavoriteArticleForUser(userID, articleID int64) (*model.ArticleInteractionState, error) {
	return nil, nil
}
func (s *immediateArticleSchedulerService) UnfavoriteArticleForUser(userID, articleID int64) (*model.ArticleInteractionState, error) {
	return nil, nil
}
func (s *immediateArticleSchedulerService) GetArticleInteractionState(userID, articleID int64) (*model.ArticleInteractionState, error) {
	return nil, nil
}
func (s *immediateArticleSchedulerService) ListArticlesByInteraction(userID int64, interactionType string, page, pageSize int) ([]*model.Article, int64, error) {
	return nil, 0, nil
}
func (s *immediateArticleSchedulerService) ToggleTop(id int64) (*model.Article, error) {
	return nil, nil
}
func (s *immediateArticleSchedulerService) UpdatePopularityScores() error { return nil }
func (s *immediateArticleSchedulerService) GetAllPublished() ([]*model.Article, error) {
	return nil, nil
}
func (s *immediateArticleSchedulerService) GetDueScheduledArticles() ([]*model.Article, error) {
	scheduledAt := time.Now().Add(-time.Minute)
	return []*model.Article{{ID: 77, Title: "due", ScheduledPublishAt: &scheduledAt}}, nil
}
func (s *immediateArticleSchedulerService) PublishScheduled(articleID int64) error {
	s.published <- articleID
	return nil
}
func (s *immediateArticleSchedulerService) SetScheduledPublishAt(articleID int64, scheduledAt *time.Time) error {
	return nil
}

func TestStartArticleScheduler_RunsImmediately(t *testing.T) {
	articleService := &immediateArticleSchedulerService{published: make(chan int64, 1)}
	runner := async.NewTaskRunner(context.Background(), zap.NewNop())
	task := articleSchedulerTask(articleService)
	if !task.enabled {
		t.Fatal("expected article scheduler task to be enabled")
	}
	if err := task.start(context.Background(), runner, zap.NewNop()); err != nil {
		t.Fatalf("expected article scheduler task to start, got %v", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = runner.Shutdown(shutdownCtx)
	}()

	select {
	case articleID := <-articleService.published:
		if articleID != 77 {
			t.Fatalf("expected article 77 to publish, got %d", articleID)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("expected article scheduler to process due articles immediately after start")
	}
}

func TestStartBackgroundTasks_ShutsDownTaskRunner(t *testing.T) {
	runner := async.NewTaskRunner(context.Background(), zap.NewNop())
	services := &appServices{taskRunner: runner}

	stop := startBackgroundTasks(&config.Config{}, zap.NewNop(), services)
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := runner.Shutdown(shutdownCtx); err == nil {
		t.Fatal("expected second shutdown to fail after startBackgroundTasks stop")
	}
}

func TestStartBackgroundTasks_StartsImmediateSchedulersThroughSharedSupervisor(t *testing.T) {
	runner := async.NewTaskRunner(context.Background(), zap.NewNop())
	uploadService := &immediateCleanupUploadService{called: make(chan time.Time, 1)}
	articleService := &immediateArticleSchedulerService{published: make(chan int64, 1)}
	services := &appServices{
		taskRunner: runner,
		upload:     uploadService,
		article:    articleService,
	}
	cfg := &config.Config{
		Upload: config.UploadConfig{
			CleanupEnabled:       true,
			CleanupIntervalHours: 24,
		},
	}

	stop := startBackgroundTasks(cfg, zap.NewNop(), services)
	defer stop()

	select {
	case <-uploadService.called:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("expected upload cleanup to run immediately")
	}

	select {
	case articleID := <-articleService.published:
		if articleID != 77 {
			t.Fatalf("expected article 77 to publish, got %d", articleID)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("expected article scheduler to run immediately")
	}
}

func TestNewBackgroundTaskSupervisor_StartsEnabledTasksOnly(t *testing.T) {
	runner := async.NewTaskRunner(context.Background(), zap.NewNop())
	supervisor := newBackgroundTaskSupervisor(context.Background(), runner, zap.NewNop())

	started := make(chan string, 2)
	supervisor.Add(backgroundTask{
		name:    "enabled",
		enabled: true,
		start: func(ctx context.Context, runner async.Runner, logger *zap.Logger) error {
			started <- "enabled"
			return nil
		},
	})
	supervisor.Add(backgroundTask{
		name:    "disabled",
		enabled: false,
		start: func(ctx context.Context, runner async.Runner, logger *zap.Logger) error {
			started <- "disabled"
			return nil
		},
	})

	supervisor.Start()
	defer supervisor.Stop()

	select {
	case name := <-started:
		if name != "enabled" {
			t.Fatalf("expected enabled task to start, got %q", name)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("expected enabled task to start")
	}

	select {
	case name := <-started:
		t.Fatalf("expected disabled task not to start, got %q", name)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestStartBackgroundTasks_StopIsSafeWithoutPerTaskStopFunctions(t *testing.T) {
	runner := async.NewTaskRunner(context.Background(), zap.NewNop())
	services := &appServices{taskRunner: runner}

	stop := startBackgroundTasks(&config.Config{Log: config.LogConfig{MaxAgeDays: 7}}, zap.NewNop(), services)
	stop()
	stop()
}
