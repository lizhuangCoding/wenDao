package model

import "time"

// ConversationRunStep 对话执行步骤模型
type ConversationRunStep struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ConversationID int64     `gorm:"not null;index:idx_conversation_run_step_conversation" json:"conversation_id"`
	RunID          int64     `gorm:"not null;index:idx_conversation_run_step_run" json:"run_id"`
	AgentName      string    `gorm:"size:64;not null" json:"agent_name"`
	Type           string    `gorm:"size:32;not null" json:"type"` // thinking, tool_use, research, log
	Summary        string    `gorm:"type:text;not null" json:"summary"`
	Detail         string    `gorm:"type:longtext" json:"detail"`
	Status         string    `gorm:"size:32;not null" json:"status"` // running, completed, failed
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// TableName 指定表名
func (ConversationRunStep) TableName() string {
	return "conversation_run_steps"
}
