package database

import (
	"gorm.io/gorm"

	"wenDao/internal/model"
)

// AutoMigrate 自动迁移表结构
func AutoMigrate(db *gorm.DB) error {
	if err := dropUserUsernameUniqueIndexes(db); err != nil {
		return err
	}

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

type userIndexRow struct {
	IndexName string `gorm:"column:INDEX_NAME"`
}

func dropUserUsernameUniqueIndexes(db *gorm.DB) error {
	if !db.Migrator().HasTable(&model.User{}) {
		return nil
	}

	var rows []userIndexRow
	if err := db.Raw(`
		SELECT INDEX_NAME
		FROM INFORMATION_SCHEMA.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE()
			AND TABLE_NAME = ?
			AND COLUMN_NAME = ?
			AND NON_UNIQUE = 0
	`, model.User{}.TableName(), "username").Scan(&rows).Error; err != nil {
		return err
	}

	dropped := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if row.IndexName == "" || row.IndexName == "PRIMARY" {
			continue
		}
		if _, exists := dropped[row.IndexName]; exists {
			continue
		}
		if err := db.Migrator().DropIndex(&model.User{}, row.IndexName); err != nil {
			return err
		}
		dropped[row.IndexName] = struct{}{}
	}

	return nil
}
