package article

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"wenDao/internal/model"
	"wenDao/internal/repository"
	"wenDao/internal/svcerrors"
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
			return nil, svcerrors.ErrArticleNotFound
		}
		return nil, fmt.Errorf("failed to get article: %w", err)
	}

	if s.taskRunner != nil {
		_ = s.taskRunner.Submit(nil, "cache article by id", func(ctx context.Context) error {
			s.setArticleToCache(article)
			return nil
		})
	} else {
		s.setArticleToCache(article)
	}
	return article, nil
}

// GetBySlug 根据 slug 获取文章
func (s *articleService) GetBySlug(slug string) (*model.Article, error) {
	article, err := s.getArticleFromSlugCache(slug)
	if err == nil && article != nil {
		return article, nil
	}

	article, err = s.articleRepo.GetBySlug(slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, svcerrors.ErrArticleNotFound
		}
		return nil, fmt.Errorf("failed to get article: %w", err)
	}

	if s.taskRunner != nil {
		_ = s.taskRunner.Submit(nil, "cache article by slug", func(ctx context.Context) error {
			s.setArticleToCache(article)
			return nil
		})
	} else {
		s.setArticleToCache(article)
	}
	return article, nil
}

// GetAllPublished 获取所有已发布的文章
func (s *articleService) GetAllPublished() ([]*model.Article, error) {
	return s.articleRepo.GetAllPublished()
}

// List 获取文章列表
func (s *articleService) List(status string, categoryID, tagID int64, keyword string, sortByPopularity bool, page, pageSize int) ([]*model.Article, int64, error) {
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
		TagID:            tagID,
		Keyword:          keyword,
		SortByPopularity: sortByPopularity,
		Page:             page,
		PageSize:         pageSize,
	}

	if articles, total, ok := s.getCachedArticleList(filter); ok {
		return articles, total, nil
	}

	cacheKey := s.articleListCacheKey(filter)
	result, err, _ := s.cacheGroup.Do(cacheKey, func() (any, error) {
		if articles, total, ok := s.getCachedArticleList(filter); ok {
			return cachedArticleList{Articles: articles, Total: total}, nil
		}

		articles, total, err := s.articleRepo.List(filter)
		if err != nil {
			return nil, err
		}
		s.setCachedArticleList(filter, articles, total)
		return cachedArticleList{Articles: articles, Total: total}, nil
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list articles: %w", err)
	}

	payload := result.(cachedArticleList)
	return payload.Articles, payload.Total, nil
}

// ListOrbitArticles 获取首页文章星球需要的轻量文章数据。
func (s *articleService) ListOrbitArticles() ([]*model.Article, error) {
	if articles, ok := s.getCachedOrbitArticles(); ok {
		return articles, nil
	}

	result, err, _ := s.cacheGroup.Do(s.articleOrbitCacheKey(), func() (any, error) {
		if articles, ok := s.getCachedOrbitArticles(); ok {
			return articles, nil
		}

		articles, err := s.articleRepo.ListOrbitArticles()
		if err != nil {
			return nil, err
		}
		if err := s.hydrateOrbitSemanticProfiles(articles); err != nil {
			return nil, fmt.Errorf("failed to hydrate orbit semantic profiles: %w", err)
		}
		s.setCachedOrbitArticles(articles)
		return articles, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list orbit articles: %w", err)
	}
	articles := result.([]*model.Article)
	return articles, nil
}

// IncrViewCount 增加文章浏览次数
func (s *articleService) IncrViewCount(id int64) error {
	return s.articleRepo.IncrementViewCount(id)
}

// LikeArticle 点赞文章
func (s *articleService) LikeArticle(id int64) error {
	article, err := s.getArticleByIDOrNotFound(id)
	if err != nil {
		return err
	}
	if err := s.articleRepo.IncrementLikeCount(id); err != nil {
		return err
	}
	s.deleteArticleFromCache(article)
	s.invalidateArticleCollections()
	return nil
}

// UnlikeArticle 取消点赞文章
func (s *articleService) UnlikeArticle(id int64) error {
	article, err := s.getArticleByIDOrNotFound(id)
	if err != nil {
		return err
	}
	if err := s.articleRepo.DecrementLikeCount(id); err != nil {
		return err
	}
	s.deleteArticleFromCache(article)
	s.invalidateArticleCollections()
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
	s.deleteArticleFromCache(article)
	s.invalidateArticleCollections()
	return article, nil
}
