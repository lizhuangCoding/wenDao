package model

import "time"

// ConversationRun 对话执行状态模型
type ConversationRun struct {
	ID                 int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	ConversationID     int64      `gorm:"not null;index:idx_conversation_run_conversation" json:"conversation_id"`
	UserID             int64      `gorm:"not null;index:idx_conversation_run_user" json:"user_id"`
	Status             string     `gorm:"size:32;not null;index:idx_conversation_run_status" json:"status"`
	CurrentStage       string     `gorm:"size:32;not null" json:"current_stage"`
	OriginalQuestion   string     `gorm:"type:text;not null" json:"original_question"`
	NormalizedQuestion string     `gorm:"type:text" json:"normalized_question"`
	PendingQuestion    *string    `gorm:"type:text" json:"pending_question,omitempty"`
	PendingContext     string     `gorm:"type:longtext" json:"pending_context"`
	LastAnswer         string     `gorm:"type:longtext" json:"last_answer"`
	LastPlan           string     `gorm:"type:longtext" json:"last_plan"`
	LastError          *string    `gorm:"type:text" json:"last_error,omitempty"`
	HeartbeatAt        *time.Time `json:"heartbeat_at,omitempty"`
	CompletedAt        *time.Time `json:"completed_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// TableName 指定表名
func (ConversationRun) TableName() string {
	return "conversation_runs"
}
