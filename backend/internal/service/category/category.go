package category

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"wenDao/internal/model"
	"wenDao/internal/repository"
)

// CategoryService 分类服务接口
type CategoryService interface {
	Create(name, slug, description string, sortOrder int) (*model.Category, error)
	GetByID(id int64) (*model.Category, error)
	GetBySlug(slug string) (*model.Category, error)
	List() ([]*model.Category, error)
	Update(id int64, name, slug, description string, sortOrder int) (*model.Category, error)
	Delete(id int64) error
}

// categoryService 分类服务实现
type categoryService struct {
	categoryRepo repository.CategoryRepository
}

// NewCategoryService 创建分类服务实例
func NewCategoryService(categoryRepo repository.CategoryRepository) CategoryService {
	return &categoryService{
		categoryRepo: categoryRepo,
	}
}

// Create 创建分类
func (s *categoryService) Create(name, slug, description string, sortOrder int) (*model.Category, error) {
	// 检查 slug 是否已存在
	existingCategory, err := s.categoryRepo.GetBySlug(slug)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check slug: %w", err)
	}
	if existingCategory != nil {
		return nil, errors.New("slug already exists")
	}

	// 创建分类
	category := &model.Category{
		Name:        name,
		Slug:        slug,
		Description: description,
		SortOrder:   sortOrder,
	}

	if err := s.categoryRepo.Create(category); err != nil {
		return nil, fmt.Errorf("failed to create category: %w", err)
	}

	return category, nil
}

// GetByID 根据 ID 获取分类
func (s *categoryService) GetByID(id int64) (*model.Category, error) {
	category, err := s.categoryRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("category not found")
		}
		return nil, fmt.Errorf("failed to get category: %w", err)
	}
	return category, nil
}

// GetBySlug 根据 slug 获取分类
func (s *categoryService) GetBySlug(slug string) (*model.Category, error) {
	category, err := s.categoryRepo.GetBySlug(slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("category not found")
		}
		return nil, fmt.Errorf("failed to get category: %w", err)
	}
	return category, nil
}

// List 获取所有分类
func (s *categoryService) List() ([]*model.Category, error) {
	categories, err := s.categoryRepo.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list categories: %w", err)
	}
	return categories, nil
}

// Update 更新分类
func (s *categoryService) Update(id int64, name, slug, description string, sortOrder int) (*model.Category, error) {
	// 检查分类是否存在
	category, err := s.categoryRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("category not found")
		}
		return nil, fmt.Errorf("failed to get category: %w", err)
	}

	// 如果修改了 slug，检查新 slug 是否已存在
	if slug != category.Slug {
		existingCategory, err := s.categoryRepo.GetBySlug(slug)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("failed to check slug: %w", err)
		}
		if existingCategory != nil {
			return nil, errors.New("slug already exists")
		}
	}

	// 更新字段
	category.Name = name
	category.Slug = slug
	category.Description = description
	category.SortOrder = sortOrder

	if err := s.categoryRepo.Update(category); err != nil {
		return nil, fmt.Errorf("failed to update category: %w", err)
	}

	return category, nil
}

// Delete 删除分类
func (s *categoryService) Delete(id int64) error {
	// 检查分类是否存在
	category, err := s.categoryRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("category not found")
		}
		return fmt.Errorf("failed to get category: %w", err)
	}

	// 检查分类下是否有文章
	if category.ArticleCount > 0 {
		return errors.New("cannot delete category with articles")
	}

	if err := s.categoryRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete category: %w", err)
	}

	return nil
}
