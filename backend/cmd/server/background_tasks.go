package main

import (
	"context"
	"time"

	"go.uber.org/zap"

	"wenDao/config"
	"wenDao/internal/pkg/async"
	"wenDao/internal/service"
)

func startBackgroundTasks(cfg *config.Config, logger *zap.Logger, services *appServices) func() {
	stopLogCleanup := startLogCleanupScheduler(cfg, logger)
	stopUploadCleanup := func() {}
	stopStatFlush := func() {}
	stopArticleScheduler := func() {}
	if services != nil {
		stopUploadCleanup = startUploadCleanupScheduler(cfg, logger, services.upload)
		stopStatFlush = startStatFlushScheduler(logger, services.stat)
		stopArticleScheduler = startArticleScheduler(logger, services.article)
	}

	return func() {
		stopLogCleanup()
		stopUploadCleanup()
		stopStatFlush()
		stopArticleScheduler()
	}
}

func startUploadCleanupScheduler(cfg *config.Config, logger *zap.Logger, uploadService service.UploadService) func() {
	if cfg == nil || logger == nil || uploadService == nil || !cfg.Upload.CleanupEnabled {
		return func() {}
	}

	interval := time.Duration(cfg.Upload.CleanupIntervalHours) * time.Hour
	if interval <= 0 {
		interval = 24 * time.Hour
	}

	ctx, cancel := context.WithCancel(context.Background())
	async.Go(ctx, logger, "upload cleanup scheduler", func(ctx context.Context) error {
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
	})

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

func startStatFlushScheduler(logger *zap.Logger, statService *service.StatService) func() {
	if logger == nil || statService == nil {
		return func() {}
	}

	ctx, cancel := context.WithCancel(context.Background())
	async.Go(ctx, logger, "stat flush scheduler", func(ctx context.Context) error {
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
	})

	return cancel
}

func startArticleScheduler(logger *zap.Logger, articleService service.ArticleService) func() {
	if logger == nil || articleService == nil {
		return func() {}
	}

	ctx, cancel := context.WithCancel(context.Background())
	async.Go(ctx, logger, "article scheduler", func(ctx context.Context) error {
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
	})

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

// startLogCleanupScheduler 每隔 24 小时清理超过 MaxAgeDays 的过期日志文件。
// 与启动时 initLogger 中执行的 pruneExpiredLogFiles 互补，确保长时间运行不重启时日志也能被清理。
func startLogCleanupScheduler(cfg *config.Config, logger *zap.Logger) func() {
	if cfg == nil || logger == nil || cfg.Log.MaxAgeDays <= 0 {
		return func() {}
	}

	ctx, cancel := context.WithCancel(context.Background())
	async.Go(ctx, logger, "log cleanup scheduler", func(ctx context.Context) error {
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
	})

	return cancel
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
