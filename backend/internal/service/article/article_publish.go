package article

import (
	"fmt"
	"time"

	"wenDao/internal/model"
	"wenDao/internal/repository"
	asyncjobrepo "wenDao/internal/repository/asyncjob"
	"wenDao/internal/svcerrors"
)

// Publish 发布文章
func (s *articleService) Publish(id int64) error {
	article, err := s.getArticleByIDOrNotFound(id)
	if err != nil {
		return err
	}

	if article.Status == "published" {
		return svcerrors.ErrArticleAlreadyPublished
	}

	article.Status = "published"
	article.AIIndexStatus = "pending"
	now := time.Now()
	article.PublishedAt = &now

	if s.writeTxRunner != nil {
		if err := s.writeTxRunner.Run(func(articleRepo repository.ArticleRepository, categoryRepo repository.CategoryRepository, jobRepo asyncjobrepo.AsyncJobRepository) error {
			if err := articleRepo.Update(article); err != nil {
				return fmt.Errorf("failed to publish article: %w", err)
			}
			if err := categoryRepo.IncrementArticleCount(article.CategoryID); err != nil {
				return fmt.Errorf("failed to increment category article count: %w", err)
			}
			return s.enqueueVectorizeJob(jobRepo, article.ID, article.Title, article.Content, article.Slug)
		}); err != nil {
			return err
		}
	} else {
		if err := s.articleRepo.Update(article); err != nil {
			return fmt.Errorf("failed to publish article: %w", err)
		}
		s.categoryRepo.IncrementArticleCount(article.CategoryID)
		s.vectorizeArticleAsync(article.ID, article.Title, article.Content, article.Slug)
	}

	s.deleteArticleFromCache(article)
	s.setArticleToCache(article)
	s.invalidateArticleCollections()
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

	if s.writeTxRunner != nil {
		if err := s.writeTxRunner.Run(func(articleRepo repository.ArticleRepository, categoryRepo repository.CategoryRepository, jobRepo asyncjobrepo.AsyncJobRepository) error {
			if err := articleRepo.Update(article); err != nil {
				return fmt.Errorf("failed to publish scheduled article: %w", err)
			}
			if err := categoryRepo.IncrementArticleCount(article.CategoryID); err != nil {
				return fmt.Errorf("failed to increment category article count: %w", err)
			}
			return s.enqueueVectorizeJob(jobRepo, article.ID, article.Title, article.Content, article.Slug)
		}); err != nil {
			return err
		}
	} else {
		if err := s.articleRepo.Update(article); err != nil {
			return fmt.Errorf("failed to publish scheduled article: %w", err)
		}
		s.categoryRepo.IncrementArticleCount(article.CategoryID)
		s.vectorizeArticleAsync(article.ID, article.Title, article.Content, article.Slug)
	}

	s.deleteArticleFromCache(article)
	s.setArticleToCache(article)
	s.invalidateArticleCollections()
	return nil
}

// Draft 转为草稿
func (s *articleService) Draft(id int64) error {
	article, err := s.getArticleByIDOrNotFound(id)
	if err != nil {
		return err
	}

	if article.Status == "draft" {
		return svcerrors.ErrArticleAlreadyDraft
	}

	article.Status = "draft"
	article.AIIndexStatus = "pending"

	if s.writeTxRunner != nil {
		if err := s.writeTxRunner.Run(func(articleRepo repository.ArticleRepository, categoryRepo repository.CategoryRepository, jobRepo asyncjobrepo.AsyncJobRepository) error {
			if err := articleRepo.Update(article); err != nil {
				return fmt.Errorf("failed to draft article: %w", err)
			}
			if err := categoryRepo.DecrementArticleCount(article.CategoryID); err != nil {
				return fmt.Errorf("failed to decrement category article count: %w", err)
			}
			return s.enqueueVectorDeleteJob(jobRepo, id)
		}); err != nil {
			return err
		}
	} else {
		if err := s.articleRepo.Update(article); err != nil {
			return fmt.Errorf("failed to draft article: %w", err)
		}
		s.categoryRepo.DecrementArticleCount(article.CategoryID)
		s.deleteArticleVectorAsync(id)
	}

	s.deleteArticleFromCache(article)
	s.setArticleToCache(article)
	s.invalidateArticleCollections()
	return nil
}
