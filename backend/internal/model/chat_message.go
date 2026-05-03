package model

import "time"

// ChatMessage 聊天消息模型
type ChatMessage struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ConversationID int64     `gorm:"not null;index:idx_conversation" json:"conversation_id"`
	RunID          *int64    `gorm:"index:idx_chat_message_run" json:"run_id,omitempty"`
	Role           string    `gorm:"size:20;not null" json:"role"` // user/assistant
	Content        string    `gorm:"type:text" json:"content"`
	CreatedAt      time.Time `json:"created_at"`

	// 关联
	Conversation *Conversation `gorm:"foreignKey:ConversationID" json:"conversation,omitempty"`
}

// TableName 指定表名
func (ChatMessage) TableName() string {
	return "chat_messages"
}
