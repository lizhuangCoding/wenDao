package database

import (
	"gorm.io/gorm"

	"wenDao/internal/model"
)

// AutoMigrate 自动迁移表结构
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{},
		&model.Category{},
		&model.Article{},
		&model.Comment{},
		&model.Upload{},
		&model.Setting{},
		&model.DailyStat{},
		&model.ArticleStat{},
		&model.Conversation{},
		&model.ChatMessage{},
		&model.ConversationMemory{},
		&model.ConversationRun{},
		&model.ConversationRunStep{},
		&model.KnowledgeDocument{},
		&model.KnowledgeDocumentSource{},
	)
}
