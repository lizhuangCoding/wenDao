package model

import "time"

const (
	KnowledgeDocumentStatusPendingReview = "pending_review"
	KnowledgeDocumentStatusApproved      = "approved"
	KnowledgeDocumentStatusRejected      = "rejected"
)

// KnowledgeDocument 知识文档模型
type KnowledgeDocument struct {
	ID               int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	Title            string     `gorm:"size:255;not null" json:"title"`
	Summary          string     `gorm:"type:text" json:"summary"`
	Content          string     `gorm:"type:longtext;not null" json:"content"`
	Status           string     `gorm:"size:32;not null;index:idx_knowledge_document_status" json:"status"`
	SourceType       string     `gorm:"size:32;not null" json:"source_type"`
	CreatedByUserID  int64      `gorm:"not null;index:idx_knowledge_document_created_by" json:"created_by_user_id"`
	ReviewedByUserID *int64     `gorm:"index:idx_knowledge_document_reviewed_by" json:"reviewed_by_user_id,omitempty"`
	ReviewedAt       *time.Time `json:"reviewed_at,omitempty"`
	ReviewNote       string     `gorm:"type:text" json:"review_note"`
	VectorizedAt     *time.Time `json:"vectorized_at,omitempty"`
	ArticleID        *int64     `gorm:"index:idx_knowledge_document_article" json:"article_id,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// TableName 指定表名
func (KnowledgeDocument) TableName() string {
	return "knowledge_documents"
}
