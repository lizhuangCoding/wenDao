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
	ListRecent(filter ConversationRunFilter) ([]model.ConversationRun, int64, error)
	Update(run *model.ConversationRun) error
	DeleteBatch(ids []int64) error
	DeleteByConversationID(conversationID int64) error
}

type ConversationRunFilter struct {
	Status   string
	Keyword  string
	Page     int
	PageSize int
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
		Where("status IN ?", []string{"running", "waiting_user"}).
		Order("updated_at DESC").
		First(&run).Error
	if err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *conversationRunRepository) ListRecent(filter ConversationRunFilter) ([]model.ConversationRun, int64, error) {
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	query := r.db.Model(&model.ConversationRun{})
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Keyword != "" {
		keyword := "%" + filter.Keyword + "%"
		query = query.Where("original_question LIKE ? OR normalized_question LIKE ? OR last_error LIKE ?", keyword, keyword, keyword)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var runs []model.ConversationRun
	err := query.Order("created_at DESC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&runs).Error
	return runs, total, err
}

func (r *conversationRunRepository) Update(run *model.ConversationRun) error {
	return r.db.Save(run).Error
}

func (r *conversationRunRepository) DeleteBatch(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Where("id IN ?", ids).Delete(&model.ConversationRun{}).Error
}

func (r *conversationRunRepository) DeleteByConversationID(conversationID int64) error {
	return r.db.Where("conversation_id = ?", conversationID).Delete(&model.ConversationRun{}).Error
}
