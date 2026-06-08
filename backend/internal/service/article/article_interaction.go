package article

import (
	"errors"
	"fmt"

	"wenDao/internal/model"
	"wenDao/internal/repository"
)

func (s *articleService) ensurePublishedArticle(articleID int64) (*model.Article, error) {
	article, err := s.getArticleByIDOrNotFound(articleID)
	if err != nil {
		return nil, err
	}
	if article.Status != "published" {
		return nil, errors.New("article not found")
	}
	return article, nil
}

func (s *articleService) LikeArticleForUser(userID, articleID int64) (*model.ArticleInteractionState, error) {
	article, err := s.ensurePublishedArticle(articleID)
	if err != nil {
		return nil, err
	}
	created, err := s.articleRepo.AddInteraction(userID, articleID, model.ArticleInteractionTypeLike)
	if err != nil {
		return nil, fmt.Errorf("failed to like article: %w", err)
	}
	if created {
		if err := s.articleRepo.IncrementLikeCount(articleID); err != nil {
			return nil, fmt.Errorf("failed to update article like count: %w", err)
		}
		s.deleteArticleFromCache(article)
		s.invalidateArticleCollections()
	}
	return s.GetArticleInteractionState(userID, articleID)
}

func (s *articleService) UnlikeArticleForUser(userID, articleID int64) (*model.ArticleInteractionState, error) {
	article, err := s.ensurePublishedArticle(articleID)
	if err != nil {
		return nil, err
	}
	removed, err := s.articleRepo.RemoveInteraction(userID, articleID, model.ArticleInteractionTypeLike)
	if err != nil {
		return nil, fmt.Errorf("failed to unlike article: %w", err)
	}
	if removed {
		if err := s.articleRepo.DecrementLikeCount(articleID); err != nil {
			return nil, fmt.Errorf("failed to update article like count: %w", err)
		}
		s.deleteArticleFromCache(article)
		s.invalidateArticleCollections()
	}
	return s.GetArticleInteractionState(userID, articleID)
}

func (s *articleService) FavoriteArticleForUser(userID, articleID int64) (*model.ArticleInteractionState, error) {
	if _, err := s.ensurePublishedArticle(articleID); err != nil {
		return nil, err
	}
	if _, err := s.articleRepo.AddInteraction(userID, articleID, model.ArticleInteractionTypeFavorite); err != nil {
		return nil, fmt.Errorf("failed to favorite article: %w", err)
	}
	return s.GetArticleInteractionState(userID, articleID)
}

func (s *articleService) UnfavoriteArticleForUser(userID, articleID int64) (*model.ArticleInteractionState, error) {
	if _, err := s.ensurePublishedArticle(articleID); err != nil {
		return nil, err
	}
	if _, err := s.articleRepo.RemoveInteraction(userID, articleID, model.ArticleInteractionTypeFavorite); err != nil {
		return nil, fmt.Errorf("failed to unfavorite article: %w", err)
	}
	return s.GetArticleInteractionState(userID, articleID)
}

func (s *articleService) GetArticleInteractionState(userID, articleID int64) (*model.ArticleInteractionState, error) {
	if _, err := s.ensurePublishedArticle(articleID); err != nil {
		return nil, err
	}
	state, err := s.articleRepo.GetInteractionState(userID, articleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get article interaction state: %w", err)
	}
	return state, nil
}

func (s *articleService) ListArticlesByInteraction(userID int64, interactionType string, page, pageSize int) ([]*model.Article, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	articles, total, err := s.articleRepo.ListByInteraction(userID, interactionType, repository.ArticleFilter{
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list interacted articles: %w", err)
	}
	return articles, total, nil
}
