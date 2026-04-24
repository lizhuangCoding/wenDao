package article

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"wenDao/internal/model"
	"wenDao/internal/repository"
)

// GetByID 根据 ID 获取文章
func (s *articleService) GetByID(id int64) (*model.Article, error) {
	article, err := s.getArticleFromCache(id)
	if err == nil && article != nil {
		return article, nil
	}

	article, err = s.articleRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("article not found")
		}
		return nil, fmt.Errorf("failed to get article: %w", err)
	}

	go s.setArticleToCache(article)
	return article, nil
}

// GetBySlug 根据 slug 获取文章
func (s *articleService) GetBySlug(slug string) (*model.Article, error) {
	article, err := s.articleRepo.GetBySlug(slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("article not found")
		}
		return nil, fmt.Errorf("failed to get article: %w", err)
	}

	go s.setArticleToCache(article)
	return article, nil
}

// List 获取文章列表
func (s *articleService) List(status string, categoryID int64, keyword string, sortByPopularity bool, page, pageSize int) ([]*model.Article, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	filter := repository.ArticleFilter{
		Status:           status,
		CategoryID:       categoryID,
		Keyword:          keyword,
		SortByPopularity: sortByPopularity,
		Page:             page,
		PageSize:         pageSize,
	}

	articles, total, err := s.articleRepo.List(filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list articles: %w", err)
	}

	return articles, total, nil
}

// IncrViewCount 增加文章浏览次数
func (s *articleService) IncrViewCount(id int64) error {
	return s.articleRepo.IncrementViewCount(id)
}

// LikeArticle 点赞文章
func (s *articleService) LikeArticle(id int64) error {
	if err := s.articleRepo.IncrementLikeCount(id); err != nil {
		return err
	}
	s.deleteArticleFromCache(id)
	return nil
}

// UnlikeArticle 取消点赞文章
func (s *articleService) UnlikeArticle(id int64) error {
	if err := s.articleRepo.DecrementLikeCount(id); err != nil {
		return err
	}
	s.deleteArticleFromCache(id)
	return nil
}

// ToggleTop 切换置顶状态
func (s *articleService) ToggleTop(id int64) (*model.Article, error) {
	article, err := s.articleRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	newTopStatus := !article.IsTop
	if err := s.articleRepo.UpdateTop(id, newTopStatus); err != nil {
		return nil, err
	}

	article.IsTop = newTopStatus
	s.deleteArticleFromCache(id)
	return article, nil
}
