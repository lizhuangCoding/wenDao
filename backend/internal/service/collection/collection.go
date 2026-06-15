package collection

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"wenDao/internal/model"
	"wenDao/internal/repository"
	"wenDao/internal/svcerrors"
)

type CollectionService interface {
	Create(name, slug, description string, sortOrder int, status string) (*model.Collection, error)
	GetByID(id int64) (*model.Collection, error)
	GetBySlug(slug string) (*model.Collection, error)
	List() ([]*model.Collection, error)
	ListPaginated(filter repository.CollectionFilter) ([]*model.Collection, int64, error)
	Update(id int64, name, slug, description string, sortOrder int, status string) (*model.Collection, error)
	Delete(id int64) error
	DeleteBatch(ids []int64) error
	SetPrimaryArticlePlacement(articleID int64, collectionID *int64, position int) error
	HydrateArticleCollectionData(article *model.Article, includeNavigation bool) error
}

type collectionService struct {
	collectionRepo repository.CollectionRepository
	articleRepo    repository.ArticleRepository
}

func NewCollectionService(collectionRepo repository.CollectionRepository, articleRepo repository.ArticleRepository) CollectionService {
	return &collectionService{collectionRepo: collectionRepo, articleRepo: articleRepo}
}

func normalizeStatus(status string) string {
	if status == "" {
		return "active"
	}
	return status
}

func (s *collectionService) Create(name, slug, description string, sortOrder int, status string) (*model.Collection, error) {
	if status = normalizeStatus(status); status != "active" && status != "hidden" {
		return nil, svcerrors.ErrInvalidCollectionStatus
	}
	existing, err := s.collectionRepo.GetBySlug(slug)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check slug: %w", err)
	}
	if existing != nil {
		return nil, svcerrors.ErrSlugAlreadyExists
	}
	collection := &model.Collection{
		Name:        name,
		Slug:        slug,
		Description: description,
		SortOrder:   sortOrder,
		Status:      status,
	}
	if err := s.collectionRepo.Create(collection); err != nil {
		return nil, fmt.Errorf("failed to create collection: %w", err)
	}
	return collection, nil
}

func (s *collectionService) GetByID(id int64) (*model.Collection, error) {
	collection, err := s.collectionRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, svcerrors.ErrCollectionNotFound
		}
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}
	return collection, nil
}

func (s *collectionService) GetBySlug(slug string) (*model.Collection, error) {
	collection, err := s.collectionRepo.GetBySlug(slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, svcerrors.ErrCollectionNotFound
		}
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}
	return collection, nil
}

func (s *collectionService) List() ([]*model.Collection, error) {
	collections, err := s.collectionRepo.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}
	return collections, nil
}

func (s *collectionService) ListPaginated(filter repository.CollectionFilter) ([]*model.Collection, int64, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	collections, total, err := s.collectionRepo.ListPaginated(filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list collections: %w", err)
	}
	return collections, total, nil
}

func (s *collectionService) Update(id int64, name, slug, description string, sortOrder int, status string) (*model.Collection, error) {
	if status = normalizeStatus(status); status != "active" && status != "hidden" {
		return nil, svcerrors.ErrInvalidCollectionStatus
	}
	collection, err := s.collectionRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, svcerrors.ErrCollectionNotFound
		}
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}
	if slug != collection.Slug {
		existing, err := s.collectionRepo.GetBySlug(slug)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("failed to check slug: %w", err)
		}
		if existing != nil {
			return nil, svcerrors.ErrSlugAlreadyExists
		}
	}
	collection.Name = name
	collection.Slug = slug
	collection.Description = description
	collection.SortOrder = sortOrder
	collection.Status = status
	if err := s.collectionRepo.Update(collection); err != nil {
		return nil, fmt.Errorf("failed to update collection: %w", err)
	}
	return collection, nil
}

func (s *collectionService) Delete(id int64) error {
	collection, err := s.collectionRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return svcerrors.ErrCollectionNotFound
		}
		return fmt.Errorf("failed to get collection: %w", err)
	}
	if collection.ArticleCount > 0 {
		return svcerrors.ErrCannotDeleteCollectionWithArticles
	}
	if err := s.collectionRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}
	return nil
}

func (s *collectionService) DeleteBatch(ids []int64) error {
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return fmt.Errorf("invalid collection id: %d", id)
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		if err := s.Delete(id); err != nil {
			return fmt.Errorf("failed to delete collection %d: %w", id, err)
		}
	}
	return nil
}

func (s *collectionService) SetPrimaryArticlePlacement(articleID int64, collectionID *int64, position int) error {
	if _, err := s.articleRepo.GetByID(articleID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return svcerrors.ErrArticleNotFound
		}
		return fmt.Errorf("failed to get article: %w", err)
	}
	if collectionID != nil && *collectionID > 0 {
		if _, err := s.collectionRepo.GetByID(*collectionID); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return svcerrors.ErrCollectionNotFound
			}
			return fmt.Errorf("failed to get collection: %w", err)
		}
	}
	return s.collectionRepo.SetPrimaryArticlePlacement(articleID, collectionID, position)
}

func (s *collectionService) HydrateArticleCollectionData(article *model.Article, includeNavigation bool) error {
	if article == nil {
		return nil
	}
	placement, err := s.collectionRepo.GetPrimaryArticlePlacement(article.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("failed to get article collection placement: %w", err)
	}
	if placement.Collection != nil {
		article.CollectionMembership = &model.ArticleCollectionMembership{
			CollectionID: placement.CollectionID,
			Name:         placement.Collection.Name,
			Slug:         placement.Collection.Slug,
			Position:     placement.Position,
		}
	}
	if includeNavigation {
		navigation, err := s.collectionRepo.GetArticleNavigation(article.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return fmt.Errorf("failed to get article collection navigation: %w", err)
		}
		article.CollectionNavigation = navigation
	}
	return nil
}
