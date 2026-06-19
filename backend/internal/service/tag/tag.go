package tag

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"wenDao/internal/model"
	"wenDao/internal/repository"
	"wenDao/internal/svcerrors"
)

// TagService 标签服务接口
type TagService interface {
	Create(name, slug string) (*model.Tag, error)
	GetByID(id int64) (*model.Tag, error)
	GetBySlug(slug string) (*model.Tag, error)
	List() ([]*model.Tag, error)
	ListPaginated(filter repository.TagFilter) ([]*model.Tag, int64, error)
	Update(id int64, name, slug string) (*model.Tag, error)
	Delete(id int64) error
	DeleteBatch(ids []int64) error
}

type tagService struct {
	tagRepo repository.TagRepository
}

func NewTagService(tagRepo repository.TagRepository) TagService {
	return &tagService{tagRepo: tagRepo}
}

func (s *tagService) Create(name, slug string) (*model.Tag, error) {
	existingTag, err := s.tagRepo.GetBySlug(slug)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check slug: %w", err)
	}
	if existingTag != nil {
		return nil, svcerrors.ErrSlugAlreadyExists
	}

	tag := &model.Tag{Name: name, Slug: slug}
	if err := s.tagRepo.Create(tag); err != nil {
		return nil, fmt.Errorf("failed to create tag: %w", err)
	}
	return tag, nil
}

func (s *tagService) GetByID(id int64) (*model.Tag, error) {
	tag, err := s.tagRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, svcerrors.ErrTagNotFound
		}
		return nil, fmt.Errorf("failed to get tag: %w", err)
	}
	return tag, nil
}

func (s *tagService) GetBySlug(slug string) (*model.Tag, error) {
	tag, err := s.tagRepo.GetBySlug(slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, svcerrors.ErrTagNotFound
		}
		return nil, fmt.Errorf("failed to get tag: %w", err)
	}
	return tag, nil
}

func (s *tagService) List() ([]*model.Tag, error) {
	tags, err := s.tagRepo.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}
	return tags, nil
}

func (s *tagService) ListPaginated(filter repository.TagFilter) ([]*model.Tag, int64, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}
	tags, total, err := s.tagRepo.ListPaginated(filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tags: %w", err)
	}
	return tags, total, nil
}

func (s *tagService) Update(id int64, name, slug string) (*model.Tag, error) {
	tag, err := s.tagRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, svcerrors.ErrTagNotFound
		}
		return nil, fmt.Errorf("failed to get tag: %w", err)
	}

	if slug != tag.Slug {
		existingTag, err := s.tagRepo.GetBySlug(slug)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("failed to check slug: %w", err)
		}
		if existingTag != nil {
			return nil, svcerrors.ErrSlugAlreadyExists
		}
	}

	tag.Name = name
	tag.Slug = slug
	if err := s.tagRepo.Update(tag); err != nil {
		return nil, fmt.Errorf("failed to update tag: %w", err)
	}
	return tag, nil
}

func (s *tagService) Delete(id int64) error {
	tag, err := s.tagRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return svcerrors.ErrTagNotFound
		}
		return fmt.Errorf("failed to get tag: %w", err)
	}
	if tag.ArticleCount > 0 {
		return svcerrors.ErrCannotDeleteTagWithArticles
	}
	if err := s.tagRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}
	return nil
}

func (s *tagService) DeleteBatch(ids []int64) error {
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return fmt.Errorf("invalid tag id: %d", id)
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		if err := s.Delete(id); err != nil {
			return fmt.Errorf("failed to delete tag %d: %w", id, err)
		}
	}
	return nil
}
