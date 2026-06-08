package article

import (
	"errors"
	"fmt"
	"time"

	"wenDao/internal/model"
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
	article.AIIndexStatus = "pending"
	now := time.Now()
	article.PublishedAt = &now

	if err := s.articleRepo.Update(article); err != nil {
		return fmt.Errorf("failed to publish article: %w", err)
	}

	s.categoryRepo.IncrementArticleCount(article.CategoryID)
	s.deleteArticleFromCache(article)
	s.setArticleToCache(article)
	s.invalidateArticleCollections()
	s.vectorizeArticleAsync(article.ID, article.Title, article.Content, article.Slug)
	return nil
}

// GetDueScheduledArticles 获取到期的定时发布文章
func (s *articleService) GetDueScheduledArticles() ([]*model.Article, error) {
	return s.articleRepo.GetDueScheduledArticles()
}

// PublishScheduled 发布定时文章
func (s *articleService) PublishScheduled(articleID int64) error {
	article, err := s.getArticleByIDOrNotFound(articleID)
	if err != nil {
		return err
	}

	if article.Status != "draft" {
		return fmt.Errorf("article %d is not in draft status", articleID)
	}

	if article.ScheduledPublishAt == nil {
		return fmt.Errorf("article %d has no scheduled publish time", articleID)
	}

	now := time.Now()
	article.Status = "published"
	article.AIIndexStatus = "pending"
	article.PublishedAt = &now
	article.ScheduledPublishAt = nil

	if err := s.articleRepo.Update(article); err != nil {
		return fmt.Errorf("failed to publish scheduled article: %w", err)
	}

	s.categoryRepo.IncrementArticleCount(article.CategoryID)
	s.deleteArticleFromCache(article)
	s.setArticleToCache(article)
	s.invalidateArticleCollections()
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
	article.AIIndexStatus = "pending"

	if err := s.articleRepo.Update(article); err != nil {
		return fmt.Errorf("failed to draft article: %w", err)
	}

	s.categoryRepo.DecrementArticleCount(article.CategoryID)
	s.deleteArticleFromCache(article)
	s.setArticleToCache(article)
	s.invalidateArticleCollections()
	s.deleteArticleVectorAsync(id)
	return nil
}
