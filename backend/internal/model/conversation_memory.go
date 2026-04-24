package model

import "time"

// ConversationMemory stores compressed, reusable context for a conversation.
type ConversationMemory struct {
	ID                   int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ConversationID       int64     `gorm:"not null;index:idx_conversation_memory_conversation" json:"conversation_id"`
	UserID               int64     `gorm:"not null;index:idx_conversation_memory_user" json:"user_id"`
	Scope                string    `gorm:"size:32;not null;index:idx_conversation_memory_scope" json:"scope"`
	Content              string    `gorm:"type:text;not null" json:"content"`
	SourceMessageIDStart int64     `gorm:"not null;default:0" json:"source_message_id_start"`
	SourceMessageIDEnd   int64     `gorm:"not null;default:0" json:"source_message_id_end"`
	Importance           int       `gorm:"not null;default:1" json:"importance"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// TableName 指定表名
func (ConversationMemory) TableName() string {
	return "conversation_memories"
}
