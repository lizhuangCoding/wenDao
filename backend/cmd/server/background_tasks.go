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

func startBackgroundTasks(cfg *config.Config, logger *zap.Logger, services *appServices) func() {
	var runner async.Runner
	stopLogCleanup := startLogCleanupScheduler(cfg, logger)
	stopUploadCleanup := func() {}
	stopStatFlush := func() {}
	stopArticleScheduler := func() {}
	stopAsyncJobWorker := func() {}
	if services != nil {
		runner = services.taskRunner
		stopUploadCleanup = startUploadCleanupScheduler(cfg, logger, runner, services.upload)
		stopStatFlush = startStatFlushScheduler(logger, runner, services.stat)
		stopArticleScheduler = startArticleScheduler(logger, runner, services.article)
		stopAsyncJobWorker = startAsyncJobWorker(logger, runner, services.asyncJob)
	}

	return func() {
		stopLogCleanup()
		stopUploadCleanup()
		stopStatFlush()
		stopArticleScheduler()
		stopAsyncJobWorker()
		if runner != nil {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := runner.Shutdown(shutdownCtx); err != nil && !errors.Is(err, async.ErrTaskRunnerClosed) {
				logger.Warn("Task runner shutdown failed", zap.Error(err))
			}
		}
	}
}

func startUploadCleanupScheduler(cfg *config.Config, logger *zap.Logger, runner async.Runner, uploadService service.UploadService) func() {
	if cfg == nil || logger == nil || runner == nil || uploadService == nil || !cfg.Upload.CleanupEnabled {
		return func() {}
	}

	interval := time.Duration(cfg.Upload.CleanupIntervalHours) * time.Hour
	if interval <= 0 {
		interval = 24 * time.Hour
	}

	ctx, cancel := context.WithCancel(context.Background())
	if err := runner.Submit(ctx, "upload cleanup scheduler", func(ctx context.Context) error {
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
	}, async.WithTimeout(0)); err != nil {
		logger.Warn("Failed to start upload cleanup scheduler", zap.Error(err))
		cancel()
		return func() {}
	}

	return cancel
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

func startStatFlushScheduler(logger *zap.Logger, runner async.Runner, statService *service.StatService) func() {
	if logger == nil || runner == nil || statService == nil {
		return func() {}
	}

	ctx, cancel := context.WithCancel(context.Background())
	if err := runner.Submit(ctx, "stat flush scheduler", func(ctx context.Context) error {
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
	}, async.WithTimeout(0)); err != nil {
		logger.Warn("Failed to start stat flush scheduler", zap.Error(err))
		cancel()
		return func() {}
	}

	return cancel
}

func startArticleScheduler(logger *zap.Logger, runner async.Runner, articleService service.ArticleService) func() {
	if logger == nil || runner == nil || articleService == nil {
		return func() {}
	}

	ctx, cancel := context.WithCancel(context.Background())
	if err := runner.Submit(ctx, "article scheduler", func(ctx context.Context) error {
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
	}, async.WithTimeout(0)); err != nil {
		logger.Warn("Failed to start article scheduler", zap.Error(err))
		cancel()
		return func() {}
	}

	return cancel
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

func runStatFlush(logger *zap.Logger, statService *service.StatService) {
	if err := statService.FlushRecentDailyStatCounters(); err != nil {
		logger.Warn("Stat flush failed", zap.Error(err))
	}
}

func startAsyncJobWorker(logger *zap.Logger, runner async.Runner, asyncJobService service.AsyncJobService) func() {
	if logger == nil || runner == nil || asyncJobService == nil {
		return func() {}
	}

	ctx, cancel := context.WithCancel(context.Background())
	if err := runner.Submit(ctx, "async job worker", func(ctx context.Context) error {
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
	}, async.WithTimeout(0)); err != nil {
		logger.Warn("Failed to start async job worker", zap.Error(err))
		cancel()
		return func() {}
	}

	return cancel
}

func runAsyncJobsOnce(ctx context.Context, logger *zap.Logger, asyncJobService service.AsyncJobService) {
	if err := asyncJobService.ProcessPending(ctx, 20); err != nil {
		logger.Warn("Async job worker failed", zap.Error(err))
	}
}

// startLogCleanupScheduler 每隔 24 小时清理超过 MaxAgeDays 的过期日志文件。
// 与启动时 initLogger 中执行的 pruneExpiredLogFiles 互补，确保长时间运行不重启时日志也能被清理。
func startLogCleanupScheduler(cfg *config.Config, logger *zap.Logger) func() {
	if cfg == nil || logger == nil || cfg.Log.MaxAgeDays <= 0 {
		return func() {}
	}

	ctx, cancel := context.WithCancel(context.Background())
	runner := async.NewTaskRunner(context.Background(), logger, async.WithDefaultTimeout(0))
	if err := runner.Submit(ctx, "log cleanup scheduler", func(ctx context.Context) error {
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
	}, async.WithTimeout(0)); err != nil {
		cancel()
		return func() {}
	}

	return func() {
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		_ = runner.Shutdown(shutdownCtx)
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
