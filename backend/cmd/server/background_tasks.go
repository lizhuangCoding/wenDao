package main

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"

	"wenDao/config"
	"wenDao/internal/pkg/async"
	"wenDao/internal/service"
)

type backgroundTask struct {
	name    string
	enabled bool
	start   func(ctx context.Context, runner async.Runner, logger *zap.Logger) error
}

type backgroundTaskSupervisor struct {
	ctx     context.Context
	cancel  context.CancelFunc
	runner  async.Runner
	logger  *zap.Logger
	tasks   []backgroundTask
	stopped bool
}

func newBackgroundTaskSupervisor(parent context.Context, runner async.Runner, logger *zap.Logger) *backgroundTaskSupervisor {
	if parent == nil {
		parent = context.Background()
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	ctx, cancel := context.WithCancel(parent)
	return &backgroundTaskSupervisor{
		ctx:    ctx,
		cancel: cancel,
		runner: runner,
		logger: logger,
	}
}

func (s *backgroundTaskSupervisor) Add(task backgroundTask) {
	if s == nil {
		return
	}
	s.tasks = append(s.tasks, task)
}

func (s *backgroundTaskSupervisor) Start() {
	if s == nil || s.runner == nil {
		return
	}
	for _, task := range s.tasks {
		if !task.enabled || task.start == nil {
			continue
		}
		if err := task.start(s.ctx, s.runner, s.logger); err != nil {
			s.logger.Warn("Failed to start background task", zap.String("task", task.name), zap.Error(err))
		}
	}
}

func (s *backgroundTaskSupervisor) Stop() {
	if s == nil || s.stopped {
		return
	}
	s.stopped = true
	if s.cancel != nil {
		s.cancel()
	}
}

func startBackgroundTasks(cfg *config.Config, logger *zap.Logger, services *appServices) func() {
	if services == nil || services.taskRunner == nil {
		return func() {}
	}

	supervisor := newBackgroundTaskSupervisor(context.Background(), services.taskRunner, logger)
	for _, task := range backgroundTasks(cfg, services) {
		supervisor.Add(task)
	}
	supervisor.Start()

	return func() {
		supervisor.Stop()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := services.taskRunner.Shutdown(shutdownCtx); err != nil && !errors.Is(err, async.ErrTaskRunnerClosed) {
			logger.Warn("Task runner shutdown failed", zap.Error(err))
		}
	}
}

func backgroundTasks(cfg *config.Config, services *appServices) []backgroundTask {
	if services == nil {
		return nil
	}
	return []backgroundTask{
		uploadCleanupTask(cfg, services.upload),
		statFlushTask(services.stat),
		articleSchedulerTask(services.article),
		asyncJobWorkerTask(services.asyncJob),
		logCleanupTask(cfg),
	}
}

func uploadCleanupTask(cfg *config.Config, uploadService service.UploadService) backgroundTask {
	if cfg == nil || uploadService == nil || !cfg.Upload.CleanupEnabled {
		return backgroundTask{}
	}

	interval := time.Duration(cfg.Upload.CleanupIntervalHours) * time.Hour
	if interval <= 0 {
		interval = 24 * time.Hour
	}

	return backgroundTask{
		name:    "upload cleanup scheduler",
		enabled: true,
		start: func(ctx context.Context, runner async.Runner, logger *zap.Logger) error {
			return runner.Submit(ctx, "upload cleanup scheduler", func(ctx context.Context) error {
				ticker := time.NewTicker(interval)
				defer ticker.Stop()

				logger.Info("Upload cleanup scheduler started",
					zap.Int("retention_days", cleanupRetentionDays(cfg.Upload)),
					zap.Duration("interval", interval),
					zap.Int("batch_size", cleanupBatchSize(cfg.Upload)))

				runUploadCleanup(logger, uploadService)

				for {
					select {
					case <-ticker.C:
						runUploadCleanup(logger, uploadService)
					case <-ctx.Done():
						logger.Info("Upload cleanup scheduler stopped")
						return nil
					}
				}
			}, async.WithTimeout(0))
		},
	}
}

func runUploadCleanup(logger *zap.Logger, uploadService service.UploadService) {
	result, err := uploadService.CleanupUnreferenced(time.Now())
	if err != nil {
		logger.Warn("Upload cleanup failed", zap.Error(err))
		return
	}
	logger.Info("Upload cleanup completed",
		zap.Int("candidates", result.Candidates),
		zap.Int("deleted", result.Deleted),
		zap.Int("skipped", result.Skipped))
}

func statFlushTask(statService *service.StatService) backgroundTask {
	if statService == nil {
		return backgroundTask{}
	}

	return backgroundTask{
		name:    "stat flush scheduler",
		enabled: true,
		start: func(ctx context.Context, runner async.Runner, logger *zap.Logger) error {
			return runner.Submit(ctx, "stat flush scheduler", func(ctx context.Context) error {
				ticker := time.NewTicker(time.Minute)
				defer ticker.Stop()

				logger.Info("Stat flush scheduler started", zap.Duration("interval", time.Minute))
				runStatFlush(logger, statService)

				for {
					select {
					case <-ticker.C:
						runStatFlush(logger, statService)
					case <-ctx.Done():
						runStatFlush(logger, statService)
						logger.Info("Stat flush scheduler stopped")
						return nil
					}
				}
			}, async.WithTimeout(0))
		},
	}
}

func runStatFlush(logger *zap.Logger, statService *service.StatService) {
	if err := statService.FlushRecentDailyStatCounters(); err != nil {
		logger.Warn("Stat flush failed", zap.Error(err))
	}
}

func articleSchedulerTask(articleService service.ArticleService) backgroundTask {
	if articleService == nil {
		return backgroundTask{}
	}

	return backgroundTask{
		name:    "article scheduler",
		enabled: true,
		start: func(ctx context.Context, runner async.Runner, logger *zap.Logger) error {
			return runner.Submit(ctx, "article scheduler", func(ctx context.Context) error {
				ticker := time.NewTicker(30 * time.Second)
				defer ticker.Stop()

				logger.Info("Article scheduler started", zap.Duration("interval", 30*time.Second))
				runArticleSchedulerOnce(logger, articleService)

				for {
					select {
					case <-ctx.Done():
						logger.Info("Article scheduler stopped")
						return nil
					case <-ticker.C:
						runArticleSchedulerOnce(logger, articleService)
					}
				}
			}, async.WithTimeout(0))
		},
	}
}

func runArticleSchedulerOnce(logger *zap.Logger, articleService service.ArticleService) {
	articles, err := articleService.GetDueScheduledArticles()
	if err != nil {
		logger.Warn("Failed to get due scheduled articles", zap.Error(err))
		return
	}
	for _, article := range articles {
		if err := articleService.PublishScheduled(article.ID); err != nil {
			logger.Error("Failed to publish scheduled article",
				zap.Int64("article_id", article.ID),
				zap.Error(err))
			continue
		}
		fields := []zap.Field{
			zap.Int64("article_id", article.ID),
			zap.String("title", article.Title),
		}
		if article.ScheduledPublishAt != nil {
			fields = append(fields, zap.Time("scheduled_publish_at", *article.ScheduledPublishAt))
		}
		logger.Info("Published scheduled article", fields...)
	}
	if len(articles) > 0 {
		logger.Info("Processed scheduled articles", zap.Int("count", len(articles)))
	}
}

func asyncJobWorkerTask(asyncJobService service.AsyncJobService) backgroundTask {
	if asyncJobService == nil {
		return backgroundTask{}
	}

	return backgroundTask{
		name:    "async job worker",
		enabled: true,
		start: func(ctx context.Context, runner async.Runner, logger *zap.Logger) error {
			return runner.Submit(ctx, "async job worker", func(ctx context.Context) error {
				ticker := time.NewTicker(5 * time.Second)
				defer ticker.Stop()

				logger.Info("Async job worker started", zap.Duration("interval", 5*time.Second))
				runAsyncJobsOnce(ctx, logger, asyncJobService)

				for {
					select {
					case <-ctx.Done():
						logger.Info("Async job worker stopped")
						return nil
					case <-ticker.C:
						runAsyncJobsOnce(ctx, logger, asyncJobService)
					}
				}
			}, async.WithTimeout(0))
		},
	}
}

func runAsyncJobsOnce(ctx context.Context, logger *zap.Logger, asyncJobService service.AsyncJobService) {
	if err := asyncJobService.ProcessPending(ctx, 20); err != nil {
		logger.Warn("Async job worker failed", zap.Error(err))
	}
}

func logCleanupTask(cfg *config.Config) backgroundTask {
	if cfg == nil || cfg.Log.MaxAgeDays <= 0 {
		return backgroundTask{}
	}

	return backgroundTask{
		name:    "log cleanup scheduler",
		enabled: true,
		start: func(ctx context.Context, runner async.Runner, logger *zap.Logger) error {
			return runner.Submit(ctx, "log cleanup scheduler", func(ctx context.Context) error {
				const cleanupInterval = 24 * time.Hour
				ticker := time.NewTicker(cleanupInterval)
				defer ticker.Stop()

				logger.Info("Log cleanup scheduler started",
					zap.Int("max_age_days", cfg.Log.MaxAgeDays),
					zap.Duration("interval", cleanupInterval))

				for {
					select {
					case <-ticker.C:
						runLogCleanup(logger, cfg.Log)
					case <-ctx.Done():
						logger.Info("Log cleanup scheduler stopped")
						return nil
					}
				}
			}, async.WithTimeout(0))
		},
	}
}

func runLogCleanup(logger *zap.Logger, cfg config.LogConfig) {
	if cfg.Output == "stdout" || cfg.Output == "" {
		if err := pruneExpiredLogFiles(aiLogDir(cfg.Output), cfg.MaxAgeDays, time.Now()); err != nil {
			logger.Warn("Periodic prune of AI log files failed", zap.Error(err))
		}
	} else {
		dir := logOutputDir(cfg.Output)
		if err := pruneExpiredLogFiles(dir, cfg.MaxAgeDays, time.Now()); err != nil {
			logger.Warn("Periodic prune of log files failed", zap.Error(err))
		}
	}
}

func cleanupRetentionDays(cfg config.UploadConfig) int {
	if cfg.CleanupRetentionDays <= 0 {
		return 2
	}
	return cfg.CleanupRetentionDays
}

func cleanupBatchSize(cfg config.UploadConfig) int {
	if cfg.CleanupBatchSize <= 0 {
		return 200
	}
	return cfg.CleanupBatchSize
}
