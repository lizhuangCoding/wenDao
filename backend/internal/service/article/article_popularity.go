package article

import (
	"time"
)

// UpdatePopularityScores 批量更新所有已发布文章的活跃度分数
func (s *articleService) UpdatePopularityScores() error {
	return s.articleRepo.UpdatePopularityScores(time.Now())
}
