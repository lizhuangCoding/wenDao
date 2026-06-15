package ai

import (
	"fmt"

	"go.uber.org/zap"

	articlerepo "wenDao/internal/repository/article"
)

// SyncPublishedArticleVectors 同步已发布文章的向量索引，仅处理 pending/failed 状态的文章。
func SyncPublishedArticleVectors(articleRepo articlerepo.ArticleRepository, semanticRepo articlerepo.ArticleSemanticProfileRepository, vectorService VectorService, logger *zap.Logger) error {
	if articleRepo == nil || vectorService == nil || logger == nil {
		return nil
	}

	const pageSize = 100
	page := 1
	totalSynced := 0

	for {
		articles, total, err := articleRepo.List(articlerepo.ArticleFilter{
			Status:          "published",
			AIIndexStatuses: []string{"pending", "failed"},
			IncludeContent:  true,
			Page:            page,
			PageSize:        pageSize,
		})
		if err != nil {
			return fmt.Errorf("failed to list published articles for vector sync: %w", err)
		}
		if len(articles) == 0 {
			break
		}

		logger.Info("Syncing published article vector batch",
			zap.Int("page", page),
			zap.Int("batch_size", len(articles)),
			zap.Int64("total", total))

		for _, article := range articles {
			if article == nil {
				continue
			}
			if err := vectorService.VectorizeArticle(article.ID, article.Title, article.Content, article.Slug); err != nil {
				if statusErr := articleRepo.UpdateAIIndexStatus(article.ID, "failed"); statusErr != nil {
					logger.Warn("Failed to mark article vector sync as failed",
						zap.Int64("article_id", article.ID),
						zap.Error(statusErr))
				}
				return fmt.Errorf("failed to sync article %d vectors: %w", article.ID, err)
			}
			if err := articleRepo.UpdateAIIndexStatus(article.ID, "success"); err != nil {
				return fmt.Errorf("failed to mark article %d vector sync as success: %w", article.ID, err)
			}
			totalSynced++
		}

		page++
	}

	logger.Info("Published article vector sync completed", zap.Int("article_count", totalSynced))
	if semanticRepo != nil {
		if err := SyncPublishedArticlesMissingSemanticProfiles(articleRepo, semanticRepo, vectorService, logger); err != nil {
			return err
		}
	}
	return nil
}

// SyncPublishedArticlesMissingSemanticProfiles 同步缺少语义画像的已发布文章。
func SyncPublishedArticlesMissingSemanticProfiles(articleRepo articlerepo.ArticleRepository, semanticRepo articlerepo.ArticleSemanticProfileRepository, vectorService VectorService, logger *zap.Logger) error {
	const pageSize = 100
	page := 1
	totalSynced := 0

	for {
		articles, _, err := articleRepo.List(articlerepo.ArticleFilter{
			Status:         "published",
			IncludeContent: true,
			Page:           page,
			PageSize:       pageSize,
		})
		if err != nil {
			return fmt.Errorf("failed to list published articles for semantic profile sync: %w", err)
		}
		if len(articles) == 0 {
			break
		}

		articleIDs := make([]int64, 0, len(articles))
		for _, article := range articles {
			if article != nil {
				articleIDs = append(articleIDs, article.ID)
			}
		}
		profilesByArticleID, err := semanticRepo.ListByArticleIDs(articleIDs)
		if err != nil {
			return fmt.Errorf("failed to list article semantic profiles: %w", err)
		}

		for _, article := range articles {
			if article == nil {
				continue
			}
			if _, exists := profilesByArticleID[article.ID]; exists {
				continue
			}
			if err := vectorService.VectorizeArticle(article.ID, article.Title, article.Content, article.Slug); err != nil {
				if statusErr := articleRepo.UpdateAIIndexStatus(article.ID, "failed"); statusErr != nil {
					logger.Warn("Failed to mark article semantic profile sync as failed",
						zap.Int64("article_id", article.ID),
						zap.Error(statusErr))
				}
				return fmt.Errorf("failed to sync article %d semantic profile: %w", article.ID, err)
			}
			if err := articleRepo.UpdateAIIndexStatus(article.ID, "success"); err != nil {
				return fmt.Errorf("failed to mark article %d semantic profile sync as success: %w", article.ID, err)
			}
			totalSynced++
		}

		page++
	}

	logger.Info("Published article semantic profile sync completed", zap.Int("article_count", totalSynced))
	return nil
}
