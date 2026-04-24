package repository

import (
	"gorm.io/gorm"

	"wenDao/internal/model"
)

// CategoryRepository 分类数据访问接口
type CategoryRepository interface {
	Create(category *model.Category) error
	GetByID(id int64) (*model.Category, error)
	GetBySlug(slug string) (*model.Category, error)
	List() ([]*model.Category, error)
	Update(category *model.Category) error
	Delete(id int64) error
	IncrementArticleCount(id int64) error
	DecrementArticleCount(id int64) error
}

// categoryRepository 分类数据访问实现
type categoryRepository struct {
	db *gorm.DB
}

// NewCategoryRepository 创建分类数据访问实例
func NewCategoryRepository(db *gorm.DB) CategoryRepository {
	return &categoryRepository{db: db}
}

// Create 创建分类
func (r *categoryRepository) Create(category *model.Category) error {
	return r.db.Create(category).Error
}

// GetByID 根据 ID 查询分类
func (r *categoryRepository) GetByID(id int64) (*model.Category, error) {
	var category model.Category
	err := r.db.Where("id = ?", id).First(&category).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}

// GetBySlug 根据 slug 查询分类
func (r *categoryRepository) GetBySlug(slug string) (*model.Category, error) {
	var category model.Category
	err := r.db.Where("slug = ?", slug).First(&category).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}

// List 获取所有分类（按 sort_order 排序）
func (r *categoryRepository) List() ([]*model.Category, error) {
	var categories []*model.Category
	err := r.db.Order("sort_order ASC, created_at DESC").Find(&categories).Error
	return categories, err
}

// Update 更新分类
func (r *categoryRepository) Update(category *model.Category) error {
	return r.db.Save(category).Error
}

// Delete 删除分类
func (r *categoryRepository) Delete(id int64) error {
	return r.db.Delete(&model.Category{}, id).Error
}

// IncrementArticleCount 增加文章计数
func (r *categoryRepository) IncrementArticleCount(id int64) error {
	return r.db.Model(&model.Category{}).Where("id = ?", id).
		UpdateColumn("article_count", gorm.Expr("article_count + ?", 1)).Error
}

// DecrementArticleCount 减少文章计数
func (r *categoryRepository) DecrementArticleCount(id int64) error {
	return r.db.Model(&model.Category{}).Where("id = ?", id).
		UpdateColumn("article_count", gorm.Expr("article_count - ?", 1)).Error
}
