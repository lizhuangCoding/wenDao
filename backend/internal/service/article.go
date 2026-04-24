package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"wenDao/internal/model"
	"wenDao/internal/pkg/hash"
	"wenDao/internal/repository"
)

// ArticleService 文章服务接口
type ArticleService interface {
	Create(title, content, summary string, categoryID, authorID int64, coverImage *string, status string) (*model.Article, error)
	GetByID(id int64) (*model.Article, error)
	GetBySlug(slug string) (*model.Article, error)
	List(status string, categoryID int64, keyword string, sortByPopularity bool, page, pageSize int) ([]*model.Article, int64, error)
	Update(id int64, title, content, summary string, categoryID int64, coverImage *string) (*model.Article, error)
	Delete(id int64) error
	Publish(id int64) error
	Draft(id int64) error
	AutoSave(id int64, title, content, summary string) error
	IncrViewCount(id int64) error
	LikeArticle(id int64) error
	UnlikeArticle(id int64) error
	ToggleTop(id int64) (*model.Article, error)
	UpdatePopularityScores() error
}

// articleService 文章服务实现
type articleService struct {
	articleRepo   repository.ArticleRepository
	categoryRepo  repository.CategoryRepository
	rdb           *redis.Client
	vectorService VectorService
	logger        *zap.Logger
}

// NewArticleService 创建文章服务实例
func NewArticleService(
	articleRepo repository.ArticleRepository,
	categoryRepo repository.CategoryRepository,
	rdb *redis.Client,
	vectorService VectorService,
	logger *zap.Logger,
) ArticleService {
	return &articleService{
		articleRepo:   articleRepo,
		categoryRepo:  categoryRepo,
		rdb:           rdb,
		vectorService: vectorService,
		logger:        logger,
	}
}

func (s *articleService) updateAIIndexStatus(id int64, status string) {
	if err := s.articleRepo.UpdateAIIndexStatus(id, status); err != nil {
		s.logger.Error("Failed to update AI index status",
			zap.Int64("article_id", id),
			zap.String("status", status),
			zap.Error(err))
	}
}

func (s *articleService) vectorizeArticleAsync(id int64, title, content, slug string) {
	if s.vectorService == nil {
		return
	}

	s.updateAIIndexStatus(id, "pending")
	go func() {
		if err := s.vectorService.VectorizeArticle(id, title, content, slug); err != nil {
			s.logger.Error("Failed to vectorize article",
				zap.Int64("article_id", id),
				zap.Error(err))
			s.updateAIIndexStatus(id, "failed")
			return
		}
		s.updateAIIndexStatus(id, "success")
	}()
}

func (s *articleService) deleteArticleVectorAsync(id int64) {
	if s.vectorService == nil {
		return
	}

	go func() {
		if err := s.vectorService.DeleteArticleVector(id); err != nil {
			s.logger.Error("Failed to delete article vectors",
				zap.Int64("article_id", id),
				zap.Error(err))
		}
	}()
}

// Create 创建文章
func (s *articleService) Create(title, content, summary string, categoryID, authorID int64, coverImage *string, status string) (*model.Article, error) {
	category, err := s.categoryRepo.GetByID(categoryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("category not found")
		}
		return nil, fmt.Errorf("failed to get category: %w", err)
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

// Update 更新文章
func (s *articleService) Update(id int64, title, content, summary string, categoryID int64, coverImage *string) (*model.Article, error) {
	article, err := s.articleRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("article not found")
		}
		return nil, fmt.Errorf("failed to get article: %w", err)
	}

	oldCategoryID := article.CategoryID
	if categoryID != oldCategoryID {
		category, err := s.categoryRepo.GetByID(categoryID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("category not found")
			}
			return nil, fmt.Errorf("failed to get category: %w", err)
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
	article, err := s.articleRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("article not found")
		}
		return fmt.Errorf("failed to get article: %w", err)
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

// Publish 发布文章
func (s *articleService) Publish(id int64) error {
	article, err := s.articleRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("article not found")
		}
		return fmt.Errorf("failed to get article: %w", err)
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
	article, err := s.articleRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("article not found")
		}
		return fmt.Errorf("failed to get article: %w", err)
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

// AutoSave 自动保存文章草稿
func (s *articleService) AutoSave(id int64, title, content, summary string) error {
	article, err := s.articleRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("article not found")
		}
		return fmt.Errorf("failed to get article: %w", err)
	}

	article.Title = title
	article.Content = content
	article.Summary = summary
	article.Status = "draft" // 只要触发自动保存，必须确保是草稿状态

	if err := s.articleRepo.Update(article); err != nil {
		return fmt.Errorf("failed to auto-save article: %w", err)
	}

	// 必须清理缓存，否则前台详情页或后台重新进入时会看到旧数据
	s.deleteArticleFromCache(id)

	// 既然变成了草稿，尝试从 AI 向量索引中删除（防止前台 AI 搜到未完成的内容）
	s.updateAIIndexStatus(id, "pending")
	s.deleteArticleVectorAsync(id)

	return nil
}

// IncrViewCount 增加文章浏览次数
func (s *articleService) IncrViewCount(id int64) error {
	return s.articleRepo.IncrementViewCount(id)
}

// getArticleFromCache 从 Redis 缓存获取文章
func (s *articleService) getArticleFromCache(id int64) (*model.Article, error) {
	ctx := context.Background()
	key := fmt.Sprintf("article:detail:%d", id)

	data, err := s.rdb.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var article model.Article
	if err := json.Unmarshal([]byte(data), &article); err != nil {
		return nil, err
	}

	return &article, nil
}

// setArticleToCache 将文章保存到 Redis 缓存
func (s *articleService) setArticleToCache(article *model.Article) {
	ctx := context.Background()
	key := fmt.Sprintf("article:detail:%d", article.ID)

	data, err := json.Marshal(article)
	if err != nil {
		return
	}

	s.rdb.Set(ctx, key, data, 30*time.Minute)
}

// deleteArticleFromCache 从 Redis 删除文章缓存
func (s *articleService) deleteArticleFromCache(id int64) {
	ctx := context.Background()
	key := fmt.Sprintf("article:detail:%d", id)
	s.rdb.Del(ctx, key)
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

// UpdatePopularityScores 批量更新所有已发布文章的活跃度分数
func (s *articleService) UpdatePopularityScores() error {
	articles, err := s.articleRepo.GetAllPublished()
	if err != nil {
		return err
	}

	now := time.Now()
	for _, article := range articles {
		// 计算发布至今的小时数
		pubTime := article.CreatedAt
		if article.PublishedAt != nil {
			pubTime = *article.PublishedAt
		}

		hours := now.Sub(pubTime).Hours()
		if hours < 0 {
			hours = 0
		}

		// Hacker News 排名算法变体
		// Score = (Views * 1 + Comments * 5 + Likes * 2) / (Age + 2)^1.5
		score := (float64(article.ViewCount)*1.0 + float64(article.CommentCount)*5.0 + float64(article.LikeCount)*2.0) / math.Pow(hours+2, 1.5)

		if err := s.articleRepo.UpdatePopularity(article.ID, score); err != nil {
			s.logger.Error("Failed to update article popularity",
				zap.Int64("id", article.ID),
				zap.Error(err))
		}
	}

	return nil
}
