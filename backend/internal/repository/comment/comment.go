package comment

import (
	"strings"

	"gorm.io/gorm"

	"wenDao/internal/model"
)

// CommentFilter 评论筛选条件
type CommentFilter struct {
	Status   string
	Keyword  string
	Page     int
	PageSize int
}

// CommentRepository 评论数据访问接口
type CommentRepository interface {
	Create(comment *model.Comment) error
	GetByID(id int64) (*model.Comment, error)
	GetByArticleID(articleID int64) ([]*model.Comment, error)
	GetByArticleIDSorted(articleID int64, sort string) ([]*model.Comment, error)
	ListAll(filter CommentFilter) ([]*model.Comment, int64, error)
	Delete(id int64) error
	Restore(id int64) error
	IncrementLike(id int64) error
	DecrementLike(id int64) error
	IncrementDislike(id int64) error
	DecrementDislike(id int64) error
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
func (r *commentRepository) ListAll(filter CommentFilter) ([]*model.Comment, int64, error) {
	var comments []*model.Comment
	var total int64

	query := r.db.Model(&model.Comment{})
	if filter.Status != "" {
		query = query.Where("comments.status = ?", filter.Status)
	}
	if strings.TrimSpace(filter.Keyword) != "" {
		keyword := "%" + strings.TrimSpace(filter.Keyword) + "%"
		query = query.
			Joins("LEFT JOIN users ON users.id = comments.user_id").
			Joins("LEFT JOIN articles ON articles.id = comments.article_id").
			Where("comments.content LIKE ? OR users.username LIKE ? OR articles.title LIKE ?", keyword, keyword, keyword)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (filter.Page - 1) * filter.PageSize
	err := query.Preload("User").Preload("ReplyToUser").Preload("Article").
		Order("comments.created_at DESC").
		Offset(offset).Limit(filter.PageSize).
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

// GetByArticleIDSorted 根据文章 ID 查询评论列表（支持排序）
func (r *commentRepository) GetByArticleIDSorted(articleID int64, sort string) ([]*model.Comment, error) {
	var comments []*model.Comment
	query := r.db.Preload("User").Preload("ReplyToUser").
		Where("article_id = ? AND status = ?", articleID, "normal")

	if sort == "hottest" {
		query = query.Order("like_count DESC, created_at DESC")
	} else {
		query = query.Order("created_at ASC")
	}

	err := query.Find(&comments).Error
	return comments, err
}

// IncrementLike 增加评论点赞数
func (r *commentRepository) IncrementLike(id int64) error {
	return r.db.Model(&model.Comment{}).Where("id = ?", id).
		UpdateColumn("like_count", gorm.Expr("like_count + ?", 1)).Error
}

// DecrementLike 减少评论点赞数，避免计数低于 0
func (r *commentRepository) DecrementLike(id int64) error {
	return r.db.Model(&model.Comment{}).Where("id = ?", id).
		UpdateColumn("like_count", gorm.Expr("CASE WHEN like_count > 0 THEN like_count - 1 ELSE 0 END")).Error
}

// IncrementDislike 增加评论点踩数
func (r *commentRepository) IncrementDislike(id int64) error {
	return r.db.Model(&model.Comment{}).Where("id = ?", id).
		UpdateColumn("dislike_count", gorm.Expr("dislike_count + ?", 1)).Error
}

// DecrementDislike 减少评论点踩数，避免计数低于 0
func (r *commentRepository) DecrementDislike(id int64) error {
	return r.db.Model(&model.Comment{}).Where("id = ?", id).
		UpdateColumn("dislike_count", gorm.Expr("CASE WHEN dislike_count > 0 THEN dislike_count - 1 ELSE 0 END")).Error
}
