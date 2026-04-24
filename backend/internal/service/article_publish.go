package service

import (
	"errors"
	"fmt"
	"time"
)

// Publish 发布文章
func (s *articleService) Publish(id int64) error {
	article, err := s.getArticleByIDOrNotFound(id)
	if err != nil {
		return err
	}

	if article.Status == "published" {
		return errors.New("article is already published")
	}

	article.Status = "published"
	now := time.Now()
	article.PublishedAt = &now

	if err := s.articleRepo.Update(article); err != nil {
		return fmt.Errorf("failed to publish article: %w", err)
	}

	s.categoryRepo.IncrementArticleCount(article.CategoryID)
	s.deleteArticleFromCache(id)
	s.vectorizeArticleAsync(article.ID, article.Title, article.Content, article.Slug)
	return nil
}

// Draft 转为草稿
func (s *articleService) Draft(id int64) error {
	article, err := s.getArticleByIDOrNotFound(id)
	if err != nil {
		return err
	}

	if article.Status == "draft" {
		return errors.New("article is already draft")
	}

	article.Status = "draft"

	if err := s.articleRepo.Update(article); err != nil {
		return fmt.Errorf("failed to draft article: %w", err)
	}

	s.categoryRepo.DecrementArticleCount(article.CategoryID)
	s.deleteArticleFromCache(id)
	s.updateAIIndexStatus(id, "pending")
	s.deleteArticleVectorAsync(id)
	return nil
}
