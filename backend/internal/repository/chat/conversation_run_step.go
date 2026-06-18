package chat

import (
	"gorm.io/gorm"

	"wenDao/internal/model"
)

// ConversationRunStepRepository 对话执行步骤仓储接口
type ConversationRunStepRepository interface {
	Create(step *model.ConversationRunStep) error
	Update(step *model.ConversationRunStep) error
	GetByConversationID(conversationID int64) ([]model.ConversationRunStep, error)
	GetByRunID(runID int64) ([]model.ConversationRunStep, error)
	GetByRunIDs(runIDs []int64) ([]model.ConversationRunStep, error)
	DeleteByRunIDs(runIDs []int64) error
	DeleteByConversationID(conversationID int64) error
}

type conversationRunStepRepository struct {
	db *gorm.DB
}

// NewConversationRunStepRepository 创建对话执行步骤仓储实例
func NewConversationRunStepRepository(db *gorm.DB) ConversationRunStepRepository {
	return &conversationRunStepRepository{db: db}
}

func (r *conversationRunStepRepository) Create(step *model.ConversationRunStep) error {
	return r.db.Create(step).Error
}

func (r *conversationRunStepRepository) Update(step *model.ConversationRunStep) error {
	return r.db.Save(step).Error
}

func (r *conversationRunStepRepository) GetByConversationID(conversationID int64) ([]model.ConversationRunStep, error) {
	var steps []model.ConversationRunStep
	err := r.db.Where("conversation_id = ?", conversationID).Order("created_at ASC").Find(&steps).Error
	return steps, err
}

func (r *conversationRunStepRepository) GetByRunID(runID int64) ([]model.ConversationRunStep, error) {
	var steps []model.ConversationRunStep
	err := r.db.Where("run_id = ?", runID).Order("created_at ASC").Find(&steps).Error
	return steps, err
}

func (r *conversationRunStepRepository) GetByRunIDs(runIDs []int64) ([]model.ConversationRunStep, error) {
	if len(runIDs) == 0 {
		return []model.ConversationRunStep{}, nil
	}
	var steps []model.ConversationRunStep
	err := r.db.Where("run_id IN ?", runIDs).Order("run_id ASC, created_at ASC").Find(&steps).Error
	return steps, err
}

func (r *conversationRunStepRepository) DeleteByRunIDs(runIDs []int64) error {
	if len(runIDs) == 0 {
		return nil
	}
	return r.db.Where("run_id IN ?", runIDs).Delete(&model.ConversationRunStep{}).Error
}

func (r *conversationRunStepRepository) DeleteByConversationID(conversationID int64) error {
	return r.db.Where("conversation_id = ?", conversationID).Delete(&model.ConversationRunStep{}).Error
}
