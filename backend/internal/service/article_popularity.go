package service

import (
	"math"
	"time"

	"go.uber.org/zap"
)

// UpdatePopularityScores 批量更新所有已发布文章的活跃度分数
func (s *articleService) UpdatePopularityScores() error {
	articles, err := s.articleRepo.GetAllPublished()
	if err != nil {
		return err
	}

	now := time.Now()
	for _, article := range articles {
		pubTime := article.CreatedAt
		if article.PublishedAt != nil {
			pubTime = *article.PublishedAt
		}

		hours := now.Sub(pubTime).Hours()
		if hours < 0 {
			hours = 0
		}

		score := (float64(article.ViewCount)*1.0 + float64(article.CommentCount)*5.0 + float64(article.LikeCount)*2.0) / math.Pow(hours+2, 1.5)

		if err := s.articleRepo.UpdatePopularity(article.ID, score); err != nil {
			s.logger.Error("Failed to update article popularity",
				zap.Int64("id", article.ID),
				zap.Error(err))
		}
	}

	return nil
}
