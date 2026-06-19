package article

import (
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"

	"wenDao/internal/model"
	"wenDao/internal/repository"
	"wenDao/internal/svcerrors"
)

// ArticleService 文章服务接口
type ArticleService interface {
	Create(title, content, summary string, categoryID, authorID int64, coverImage *string, status string) (*model.Article, error)
	GetByID(id int64) (*model.Article, error)
	GetBySlug(slug string) (*model.Article, error)
	List(status string, categoryID, tagID int64, keyword string, sortByPopularity bool, page, pageSize int) ([]*model.Article, int64, error)
	SearchArticles(keyword string, categoryID, tagID int64, page, pageSize int) ([]ArticleSearchResult, int64, error)
	ListOrbitArticles() ([]*model.Article, error)
	Update(id int64, title, content, summary string, categoryID int64, coverImage *string) (*model.Article, error)
	SetTags(id int64, tagIDs []int64) (*model.Article, error)
	Delete(id int64) error
	DeleteBatch(ids []int64) error
	Publish(id int64) error
	Draft(id int64) error
	AutoSave(id int64, title, content, summary string) error
	IncrViewCount(id int64) error
	LikeArticle(id int64) error
	UnlikeArticle(id int64) error
	LikeArticleForUser(userID, articleID int64) (*model.ArticleInteractionState, error)
	UnlikeArticleForUser(userID, articleID int64) (*model.ArticleInteractionState, error)
	FavoriteArticleForUser(userID, articleID int64) (*model.ArticleInteractionState, error)
	UnfavoriteArticleForUser(userID, articleID int64) (*model.ArticleInteractionState, error)
	GetArticleInteractionState(userID, articleID int64) (*model.ArticleInteractionState, error)
	ListArticlesByInteraction(userID int64, interactionType string, page, pageSize int) ([]*model.Article, int64, error)
	ToggleTop(id int64) (*model.Article, error)
	UpdatePopularityScores() error
	GetAllPublished() ([]*model.Article, error)
	GetDueScheduledArticles() ([]*model.Article, error)
	PublishScheduled(articleID int64) error
	SetScheduledPublishAt(articleID int64, t *time.Time) error
}

// articleService 文章服务实现
type articleService struct {
	articleRepo   repository.ArticleRepository
	semanticRepo  repository.ArticleSemanticProfileRepository
	categoryRepo  repository.CategoryRepository
	tagRepo       repository.TagRepository
	cache         articleCacheStore
	vectorService VectorService
	logger        *zap.Logger
	cacheGroup    singleflight.Group
}

// NewArticleService 创建文章服务实例
func NewArticleService(
	articleRepo repository.ArticleRepository,
	categoryRepo repository.CategoryRepository,
	rdb *redis.Client,
	vectorService VectorService,
	logger *zap.Logger,
	extras ...any,
) ArticleService {
	semanticRepo, tagRepo := parseArticleServiceExtras(extras...)
	return &articleService{
		articleRepo:   articleRepo,
		semanticRepo:  semanticRepo,
		categoryRepo:  categoryRepo,
		tagRepo:       tagRepo,
		cache:         newRedisArticleCacheStore(rdb),
		vectorService: vectorService,
		logger:        logger,
	}
}

func newArticleServiceWithCacheStore(
	articleRepo repository.ArticleRepository,
	categoryRepo repository.CategoryRepository,
	cache articleCacheStore,
	vectorService VectorService,
	logger *zap.Logger,
	extras ...any,
) *articleService {
	semanticRepo, tagRepo := parseArticleServiceExtras(extras...)
	return &articleService{
		articleRepo:   articleRepo,
		semanticRepo:  semanticRepo,
		categoryRepo:  categoryRepo,
		tagRepo:       tagRepo,
		cache:         cache,
		vectorService: vectorService,
		logger:        logger,
	}
}

func parseArticleServiceExtras(extras ...any) (repository.ArticleSemanticProfileRepository, repository.TagRepository) {
	var semanticRepo repository.ArticleSemanticProfileRepository
	var tagRepo repository.TagRepository
	for _, extra := range extras {
		switch repo := extra.(type) {
		case repository.ArticleSemanticProfileRepository:
			if semanticRepo == nil {
				semanticRepo = repo
			}
		case repository.TagRepository:
			if tagRepo == nil {
				tagRepo = repo
			}
		}
	}
	return semanticRepo, tagRepo
}

func (s *articleService) getArticleByIDOrNotFound(id int64) (*model.Article, error) {
	article, err := s.articleRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, svcerrors.ErrArticleNotFound
		}
		return nil, fmt.Errorf("failed to get article: %w", err)
	}
	return article, nil
}

func (s *articleService) getCategoryByIDOrNotFound(categoryID int64) (*model.Category, error) {
	category, err := s.categoryRepo.GetByID(categoryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, svcerrors.ErrCategoryNotFound
		}
		return nil, fmt.Errorf("failed to get category: %w", err)
	}
	return category, nil
}
