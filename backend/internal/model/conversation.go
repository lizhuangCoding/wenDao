package model

import "time"

// Conversation 对话模型
type Conversation struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64     `gorm:"not null;index:idx_user" json:"user_id"`
	Title     string    `gorm:"size:255" json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// 模型选择
	ModelProvider string `gorm:"size:50" json:"model_provider"`
	ModelName     string `gorm:"size:100" json:"model_name"`

	// 分享
	ShareToken string `gorm:"uniqueIndex;size:64" json:"share_token"`
	IsShared   bool   `gorm:"default:false" json:"is_shared"`

	// 关联
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName 指定表名
func (Conversation) TableName() string {
	return "conversations"
}
