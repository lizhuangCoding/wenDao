package main

import (
	"context"
	"time"

	"go.uber.org/zap"

	"wenDao/config"
	"wenDao/internal/service"
)

func startBackgroundTasks(cfg *config.Config, logger *zap.Logger, services *appServices) func() {
	stopUploadCleanup := func() {}
	stopStatFlush := func() {}
	stopArticleScheduler := func() {}
	if services != nil {
		stopUploadCleanup = startUploadCleanupScheduler(cfg, logger, services.upload)
		stopStatFlush = startStatFlushScheduler(logger, services.stat)
		stopArticleScheduler = startArticleScheduler(logger, services.article)
	}

	return func() {
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
	go func() {
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
				return
			}
		}
	}()

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
	go func() {
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
				return
			}
		}
	}()

	return cancel
}

func startArticleScheduler(logger *zap.Logger, articleService service.ArticleService) func() {
	if logger == nil || articleService == nil {
		return func() {}
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		logger.Info("Article scheduler started", zap.Duration("interval", 30*time.Second))

		for {
			select {
			case <-ctx.Done():
				logger.Info("Article scheduler stopped")
				return
			case <-ticker.C:
				articles, err := articleService.GetDueScheduledArticles()
				if err != nil {
					logger.Warn("Failed to get due scheduled articles", zap.Error(err))
					continue
				}
				for _, article := range articles {
					if err := articleService.PublishScheduled(article.ID); err != nil {
						logger.Error("Failed to publish scheduled article",
							zap.Int64("article_id", article.ID),
							zap.Error(err))
						continue
					}
					logger.Info("Published scheduled article",
						zap.Int64("article_id", article.ID),
						zap.String("title", article.Title))
				}
				if len(articles) > 0 {
					logger.Info("Processed scheduled articles", zap.Int("count", len(articles)))
				}
			}
		}
	}()

	return cancel
}

func runStatFlush(logger *zap.Logger, statService *service.StatService) {
	if err := statService.FlushRecentDailyStatCounters(); err != nil {
		logger.Warn("Stat flush failed", zap.Error(err))
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
