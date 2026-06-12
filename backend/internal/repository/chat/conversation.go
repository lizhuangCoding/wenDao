package chat

import (
	"gorm.io/gorm"

	"wenDao/internal/model"
)

// ConversationRepository 对话数据访问接口
type ConversationRepository interface {
	Create(conv *model.Conversation) error
	GetByID(id int64) (*model.Conversation, error)
	GetByUserID(userID int64) ([]model.Conversation, error)
	GetByShareToken(token string) (*model.Conversation, error)
	Update(conv *model.Conversation) error
	UpdateShare(id int64, share bool, token string) error
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
	return r.createQuery(conv).Error
}

func (r *conversationRepository) createQuery(conv *model.Conversation) *gorm.DB {
	if conv != nil && conv.ShareToken == "" {
		return r.db.Omit("share_token").Create(conv)
	}
	return r.db.Create(conv)
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
	return r.updateQuery(conv).Error
}

func (r *conversationRepository) updateQuery(conv *model.Conversation) *gorm.DB {
	if conv != nil && conv.ShareToken == "" {
		return r.db.Omit("share_token").Save(conv)
	}
	return r.db.Save(conv)
}

// GetByShareToken 根据分享令牌查询对话
func (r *conversationRepository) GetByShareToken(token string) (*model.Conversation, error) {
	var conv model.Conversation
	err := r.db.Preload("User").Where("share_token = ? AND is_shared = ?", token, true).First(&conv).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

// UpdateShare 更新分享状态
func (r *conversationRepository) UpdateShare(id int64, share bool, token string) error {
	return r.updateShareQuery(id, share, token).Error
}

func (r *conversationRepository) updateShareQuery(id int64, share bool, token string) *gorm.DB {
	shareToken := any(token)
	if !share {
		shareToken = nil
	}
	return r.db.Model(&model.Conversation{}).Where("id = ?", id).
		Updates(map[string]any{
			"is_shared":   share,
			"share_token": shareToken,
		})
}

// Delete 删除对话
func (r *conversationRepository) Delete(id int64) error {
	return r.db.Delete(&model.Conversation{}, id).Error
}
