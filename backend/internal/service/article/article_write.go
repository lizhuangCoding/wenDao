package article

import (
	"fmt"
	"time"

	"wenDao/internal/model"
	"wenDao/internal/pkg/hash"
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

	if err := s.articleRepo.Update(article); err != nil {
		return nil, fmt.Errorf("failed to update article: %w", err)
	}

	s.deleteArticleFromCache(id)

	if article.Status == "published" {
		s.vectorizeArticleAsync(article.ID, article.Title, article.Content, article.Slug)
	} else {
		s.updateAIIndexStatus(article.ID, "pending")
	}

	return article, nil
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

	s.deleteArticleFromCache(id)
	s.deleteArticleVectorAsync(id)
	return nil
}

// DeleteBatch 批量删除文章，复用单条删除的依赖清理和计数逻辑
func (s *articleService) DeleteBatch(ids []int64) error {
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return fmt.Errorf("invalid article id: %d", id)
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		if err := s.Delete(id); err != nil {
			return fmt.Errorf("failed to delete article %d: %w", id, err)
		}
	}
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

	if err := s.articleRepo.Update(article); err != nil {
		return fmt.Errorf("failed to auto-save article: %w", err)
	}

	s.deleteArticleFromCache(id)
	s.updateAIIndexStatus(id, "pending")
	s.deleteArticleVectorAsync(id)
	return nil
}
