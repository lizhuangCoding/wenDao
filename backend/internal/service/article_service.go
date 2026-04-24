package service

import (
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"wenDao/internal/model"
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

func (s *articleService) getArticleByIDOrNotFound(id int64) (*model.Article, error) {
	article, err := s.articleRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("article not found")
		}
		return nil, fmt.Errorf("failed to get article: %w", err)
	}
	return article, nil
}

func (s *articleService) getCategoryByIDOrNotFound(categoryID int64) (*model.Category, error) {
	category, err := s.categoryRepo.GetByID(categoryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("category not found")
		}
		return nil, fmt.Errorf("failed to get category: %w", err)
	}
	return category, nil
}
