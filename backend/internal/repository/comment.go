package repository

import (
	"gorm.io/gorm"

	"wenDao/internal/model"
)

// CommentRepository 评论数据访问接口
type CommentRepository interface {
	Create(comment *model.Comment) error
	GetByID(id int64) (*model.Comment, error)
	GetByArticleID(articleID int64) ([]*model.Comment, error)
	ListAll(page, pageSize int) ([]*model.Comment, int64, error)
	Delete(id int64) error
	Restore(id int64) error
}

// commentRepository 评论数据访问实现
type commentRepository struct {
	db *gorm.DB
}

// NewCommentRepository 创建评论数据访问实例
func NewCommentRepository(db *gorm.DB) CommentRepository {
	return &commentRepository{db: db}
}

// Create 创建评论
func (r *commentRepository) Create(comment *model.Comment) error {
	return r.db.Create(comment).Error
}

// GetByID 根据 ID 查询评论（预加载用户信息）
func (r *commentRepository) GetByID(id int64) (*model.Comment, error) {
	var comment model.Comment
	err := r.db.Preload("User").Preload("ReplyToUser").Where("id = ?", id).First(&comment).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

// GetByArticleID 根据文章 ID 查询评论列表（预加载用户信息，按时间正序）
func (r *commentRepository) GetByArticleID(articleID int64) ([]*model.Comment, error) {
	var comments []*model.Comment
	err := r.db.Preload("User").Preload("ReplyToUser").
		Where("article_id = ? AND status = ?", articleID, "normal").
		Order("created_at ASC").
		Find(&comments).Error
	return comments, err
}

// ListAll 获取所有评论列表（管理员，包含软删除的）
func (r *commentRepository) ListAll(page, pageSize int) ([]*model.Comment, int64, error) {
	var comments []*model.Comment
	var total int64

	query := r.db.Model(&model.Comment{})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	err := query.Preload("User").Preload("ReplyToUser").Preload("Article").
		Order("created_at DESC").
		Offset(offset).Limit(pageSize).
		Find(&comments).Error

	return comments, total, err
}

// Delete 删除评论（软删除，修改状态）
func (r *commentRepository) Delete(id int64) error {
	return r.db.Model(&model.Comment{}).Where("id = ?", id).
		Update("status", "deleted").Error
}

// Restore 恢复评论（将状态改回 normal）
func (r *commentRepository) Restore(id int64) error {
	return r.db.Model(&model.Comment{}).Where("id = ?", id).
		Update("status", "normal").Error
}
