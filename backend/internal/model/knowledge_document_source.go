package model

import "time"

// KnowledgeDocumentSource 知识文档来源模型
type KnowledgeDocumentSource struct {
	ID                  int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	KnowledgeDocumentID int64     `gorm:"not null;index:idx_kd_source_document" json:"knowledge_document_id"`
	SourceURL           string    `gorm:"type:text;not null" json:"source_url"`
	SourceTitle         string    `gorm:"size:500" json:"source_title"`
	SourceDomain        string    `gorm:"size:255" json:"source_domain"`
	SourceSnippet       string    `gorm:"type:text" json:"source_snippet"`
	SortOrder           int       `gorm:"not null" json:"sort_order"`
	CreatedAt           time.Time `json:"created_at"`
}

// TableName 指定表名
func (KnowledgeDocumentSource) TableName() string {
	return "knowledge_document_sources"
}
