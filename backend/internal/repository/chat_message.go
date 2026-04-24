package repository

import (
	"gorm.io/gorm"

	"wenDao/internal/model"
)

// ChatMessageRepository 聊天消息数据访问接口
type ChatMessageRepository interface {
	Create(msg *model.ChatMessage) error
	GetByConversationID(conversationID int64) ([]model.ChatMessage, error)
	DeleteByConversationID(conversationID int64) error
}

// chatMessageRepository 聊天消息数据访问实现
type chatMessageRepository struct {
	db *gorm.DB
}

// NewChatMessageRepository 创建聊天消息数据访问实例
func NewChatMessageRepository(db *gorm.DB) ChatMessageRepository {
	return &chatMessageRepository{db: db}
}

// Create 创建聊天消息
func (r *chatMessageRepository) Create(msg *model.ChatMessage) error {
	return r.db.Create(msg).Error
}

// GetByConversationID 根据对话 ID 查询消息列表（按时间正序）
func (r *chatMessageRepository) GetByConversationID(conversationID int64) ([]model.ChatMessage, error) {
	var msgs []model.ChatMessage
	err := r.db.Where("conversation_id = ?", conversationID).
		Order("created_at ASC").
		Find(&msgs).Error
	return msgs, err
}

// DeleteByConversationID 根据对话 ID 删除消息
func (r *chatMessageRepository) DeleteByConversationID(conversationID int64) error {
	return r.db.Where("conversation_id = ?", conversationID).
		Delete(&model.ChatMessage{}).Error
}