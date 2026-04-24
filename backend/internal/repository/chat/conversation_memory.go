package chat

import (
	"errors"

	"gorm.io/gorm"

	"wenDao/internal/model"
)

// ConversationMemoryRepository 对话记忆仓储接口
type ConversationMemoryRepository interface {
	Upsert(memory *model.ConversationMemory) error
	GetByConversationID(conversationID int64) ([]model.ConversationMemory, error)
	GetByConversationIDAndScope(conversationID int64, scope string) (*model.ConversationMemory, error)
	DeleteByConversationID(conversationID int64) error
}

type conversationMemoryRepository struct {
	db *gorm.DB
}

// NewConversationMemoryRepository 创建对话记忆仓储实例
func NewConversationMemoryRepository(db *gorm.DB) ConversationMemoryRepository {
	return &conversationMemoryRepository{db: db}
}

func (r *conversationMemoryRepository) Upsert(memory *model.ConversationMemory) error {
	if memory == nil {
		return nil
	}
	if memory.ID != 0 {
		return r.db.Save(memory).Error
	}
	existing, err := r.GetByConversationIDAndScope(memory.ConversationID, memory.Scope)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if existing != nil {
		memory.ID = existing.ID
		memory.CreatedAt = existing.CreatedAt
	}
	return r.db.Save(memory).Error
}

func (r *conversationMemoryRepository) GetByConversationID(conversationID int64) ([]model.ConversationMemory, error) {
	var memories []model.ConversationMemory
	err := r.db.Where("conversation_id = ?", conversationID).
		Order("importance DESC, updated_at DESC").
		Find(&memories).Error
	return memories, err
}

func (r *conversationMemoryRepository) GetByConversationIDAndScope(conversationID int64, scope string) (*model.ConversationMemory, error) {
	var memory model.ConversationMemory
	err := r.db.Where("conversation_id = ? AND scope = ?", conversationID, scope).
		Order("updated_at DESC").
		First(&memory).Error
	if err != nil {
		return nil, err
	}
	return &memory, nil
}

func (r *conversationMemoryRepository) DeleteByConversationID(conversationID int64) error {
	return r.db.Where("conversation_id = ?", conversationID).Delete(&model.ConversationMemory{}).Error
}
