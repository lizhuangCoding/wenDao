package article

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"wenDao/internal/model"
	"wenDao/internal/pkg/hash"
	"wenDao/internal/svcerrors"
)

// Create 创建文章
func (s *articleService) Create(title, content, summary string, categoryID, authorID int64, coverImage *string, status string) (*model.Article, error) {
	category, err := s.getCategoryByIDOrNotFound(categoryID)
	if err != nil {
		return nil, err
	}

	article := &model.Article{
		Title:         title,
		Content:       content,
		Summary:       summary,
		CategoryID:    categoryID,
		AuthorID:      authorID,
		CoverImage:    coverImage,
		Status:        status,
		AIIndexStatus: "pending",
		SourceType:    model.ArticleSourceTypeManual,
	}

	if status == "published" {
		now := time.Now()
		article.PublishedAt = &now
	}

	if err := s.articleRepo.Create(article); err != nil {
		return nil, fmt.Errorf("failed to create article: %w", err)
	}

	slug := hash.GenerateSlug(article.ID)
	if err := s.articleRepo.UpdateSlug(article.ID, slug); err != nil {
		return nil, fmt.Errorf("failed to update slug: %w", err)
	}
	article.Slug = slug

	if status == "published" {
		s.categoryRepo.IncrementArticleCount(categoryID)
		s.vectorizeArticleAsync(article.ID, article.Title, article.Content, article.Slug)
	}

	article.Category = category
	s.setArticleToCache(article)
	s.invalidateArticleCollections()
	return article, nil
}

// Update 更新文章
func (s *articleService) Update(id int64, title, content, summary string, categoryID int64, coverImage *string) (*model.Article, error) {
	article, err := s.getArticleByIDOrNotFound(id)
	if err != nil {
		return nil, err
	}

	oldCategoryID := article.CategoryID
	if categoryID != oldCategoryID {
		category, err := s.getCategoryByIDOrNotFound(categoryID)
		if err != nil {
			return nil, err
		}
		article.Category = category

		if article.Status == "published" {
			s.categoryRepo.DecrementArticleCount(oldCategoryID)
			s.categoryRepo.IncrementArticleCount(categoryID)
		}
	}

	article.Title = title
	article.Content = content
	article.Summary = summary
	article.CategoryID = categoryID
	article.CoverImage = coverImage
	if article.Status == "published" {
		article.AIIndexStatus = "pending"
	}

	if err := s.articleRepo.Update(article); err != nil {
		return nil, fmt.Errorf("failed to update article: %w", err)
	}

	s.deleteArticleFromCache(article)
	s.setArticleToCache(article)
	s.invalidateArticleCollections()

	if article.Status == "published" {
		s.vectorizeArticleAsync(article.ID, article.Title, article.Content, article.Slug)
	}

	return article, nil
}

// SetTags 设置文章标签。
func (s *articleService) SetTags(id int64, tagIDs []int64) (*model.Article, error) {
	article, err := s.getArticleByIDOrNotFound(id)
	if err != nil {
		return nil, err
	}
	if s.tagRepo == nil {
		return nil, fmt.Errorf("tag repository is not configured")
	}
	if err := s.tagRepo.SetArticleTags(id, tagIDs); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, svcerrors.ErrTagNotFound
		}
		return nil, fmt.Errorf("failed to set article tags: %w", err)
	}
	s.deleteArticleFromCache(article)
	s.invalidateArticleCollections()
	updated, err := s.articleRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get article after setting tags: %w", err)
	}
	s.setArticleToCache(updated)
	return updated, nil
}

// Delete 删除文章
func (s *articleService) Delete(id int64) error {
	article, err := s.getArticleByIDOrNotFound(id)
	if err != nil {
		return err
	}

	if err := s.articleRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete article: %w", err)
	}

	if article.Status == "published" {
		s.categoryRepo.DecrementArticleCount(article.CategoryID)
	}

	s.deleteArticleFromCache(article)
	s.invalidateArticleCollections()
	s.deleteArticleVectorAsync(id)
	return nil
}

// DeleteBatch 批量删除文章，批量清理依赖数据并刷新相关缓存。
func (s *articleService) DeleteBatch(ids []int64) error {
	seen := make(map[int64]struct{}, len(ids))
	uniqueIDs := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return fmt.Errorf("invalid article id: %d", id)
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		uniqueIDs = append(uniqueIDs, id)
	}
	if len(uniqueIDs) == 0 {
		return nil
	}

	articles, err := s.articleRepo.DeleteBatch(uniqueIDs)
	if err != nil {
		return fmt.Errorf("failed to delete articles: %w", err)
	}

	for _, article := range articles {
		s.deleteArticleFromCache(article)
		s.deleteArticleVectorAsync(article.ID)
	}
	s.invalidateArticleCollections()
	return nil
}

// AutoSave 自动保存文章草稿
func (s *articleService) AutoSave(id int64, title, content, summary string) error {
	article, err := s.getArticleByIDOrNotFound(id)
	if err != nil {
		return err
	}

	article.Title = title
	article.Content = content
	article.Summary = summary
	article.Status = "draft"
	article.AIIndexStatus = "pending"

	if err := s.articleRepo.Update(article); err != nil {
		return fmt.Errorf("failed to auto-save article: %w", err)
	}

	s.deleteArticleFromCache(article)
	s.invalidateArticleCollections()
	s.deleteArticleVectorAsync(id)
	return nil
}

// SetScheduledPublishAt 设置文章定时发布时间
func (s *articleService) SetScheduledPublishAt(articleID int64, t *time.Time) error {
	article, err := s.getArticleByIDOrNotFound(articleID)
	if err != nil {
		return err
	}
	article.ScheduledPublishAt = t
	if err := s.articleRepo.Update(article); err != nil {
		return fmt.Errorf("failed to set scheduled publish time: %w", err)
	}
	s.deleteArticleFromCache(article)
	s.setArticleToCache(article)
	s.invalidateArticleCollections()
	return nil
}
