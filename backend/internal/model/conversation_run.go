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
	Provider           string     `gorm:"size:64" json:"provider"`
	ModelName          string     `gorm:"size:128" json:"model_name"`
	PromptTokens       int64      `gorm:"not null;default:0" json:"prompt_tokens"`
	CompletionTokens   int64      `gorm:"not null;default:0" json:"completion_tokens"`
	EstimatedCost      float64    `gorm:"not null;default:0" json:"estimated_cost"`
	CostCurrency       string     `gorm:"size:16;not null;default:USD" json:"cost_currency"`
	CostStatus         string     `gorm:"size:32;not null;default:not_collected" json:"cost_status"`
	SourceQualityScore int        `gorm:"not null;default:0" json:"source_quality_score"`
	FailureCategory    string     `gorm:"size:64" json:"failure_category"`
	FailureFingerprint string     `gorm:"size:128" json:"failure_fingerprint"`
	HeartbeatAt        *time.Time `json:"heartbeat_at,omitempty"`
	CompletedAt        *time.Time `json:"completed_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// TableName 指定表名
func (ConversationRun) TableName() string {
	return "conversation_runs"
}
