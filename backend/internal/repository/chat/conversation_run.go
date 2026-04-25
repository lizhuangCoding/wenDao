package chat

import (
	"gorm.io/gorm"

	"wenDao/internal/model"
)

// ConversationRunRepository 对话执行状态数据访问接口
type ConversationRunRepository interface {
	Create(run *model.ConversationRun) error
	GetByID(id int64) (*model.ConversationRun, error)
	GetActiveByConversationID(conversationID int64) (*model.ConversationRun, error)
	Update(run *model.ConversationRun) error
	DeleteByConversationID(conversationID int64) error
}

type conversationRunRepository struct {
	db *gorm.DB
}

// NewConversationRunRepository 创建对话执行状态仓储实例
func NewConversationRunRepository(db *gorm.DB) ConversationRunRepository {
	return &conversationRunRepository{db: db}
}

func (r *conversationRunRepository) Create(run *model.ConversationRun) error {
	return r.db.Create(run).Error
}

func (r *conversationRunRepository) GetByID(id int64) (*model.ConversationRun, error) {
	var run model.ConversationRun
	if err := r.db.First(&run, id).Error; err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *conversationRunRepository) GetActiveByConversationID(conversationID int64) (*model.ConversationRun, error) {
	var run model.ConversationRun
	err := r.db.Where("conversation_id = ?", conversationID).
		Order("updated_at DESC").
		First(&run).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *conversationRunRepository) Update(run *model.ConversationRun) error {
	return r.db.Save(run).Error
}

func (r *conversationRunRepository) DeleteByConversationID(conversationID int64) error {
	return r.db.Where("conversation_id = ?", conversationID).Delete(&model.ConversationRun{}).Error
}
