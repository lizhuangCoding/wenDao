package repository

import (
	"gorm.io/gorm"

	"wenDao/internal/model"
)

// ConversationRepository 对话数据访问接口
type ConversationRepository interface {
	Create(conv *model.Conversation) error
	GetByID(id int64) (*model.Conversation, error)
	GetByUserID(userID int64) ([]model.Conversation, error)
	Update(conv *model.Conversation) error
	Delete(id int64) error
}

// conversationRepository 对话数据访问实现
type conversationRepository struct {
	db *gorm.DB
}

// NewConversationRepository 创建对话数据访问实例
func NewConversationRepository(db *gorm.DB) ConversationRepository {
	return &conversationRepository{db: db}
}

// Create 创建对话
func (r *conversationRepository) Create(conv *model.Conversation) error {
	return r.db.Create(conv).Error
}

// GetByID 根据 ID 查询对话（预加载用户信息）
func (r *conversationRepository) GetByID(id int64) (*model.Conversation, error) {
	var conv model.Conversation
	err := r.db.Preload("User").Where("id = ?", id).First(&conv).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

// GetByUserID 根据用户 ID 查询对话列表
func (r *conversationRepository) GetByUserID(userID int64) ([]model.Conversation, error) {
	var convs []model.Conversation
	err := r.db.Where("user_id = ?", userID).
		Order("updated_at DESC").
		Find(&convs).Error
	return convs, err
}

// Update 更新对话
func (r *conversationRepository) Update(conv *model.Conversation) error {
	return r.db.Save(conv).Error
}

// Delete 删除对话
func (r *conversationRepository) Delete(id int64) error {
	return r.db.Delete(&model.Conversation{}, id).Error
}