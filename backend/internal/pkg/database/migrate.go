package database

import (
	"fmt"
	"strings"

	"gorm.io/gorm"

	"wenDao/internal/model"
)

// AutoMigrate 自动迁移表结构
func AutoMigrate(db *gorm.DB) error {
	disableForeignKeyConstraintsWhenMigrating(db)

	if err := dropForeignKeyConstraints(db); err != nil {
		return err
	}

	if err := dropUserUsernameUniqueIndexes(db); err != nil {
		return err
	}

	return db.AutoMigrate(autoMigrateModels()...)
}

func autoMigrateModels() []any {
	return []any{
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
		&model.Notification{},
	}
}

type userIndexRow struct {
	IndexName string `gorm:"column:INDEX_NAME"`
}

type foreignKeyConstraintRow struct {
	TableName      string `gorm:"column:TABLE_NAME"`
	ConstraintName string `gorm:"column:CONSTRAINT_NAME"`
}

func disableForeignKeyConstraintsWhenMigrating(db *gorm.DB) {
	if db == nil || db.Config == nil {
		return
	}
	db.Config.DisableForeignKeyConstraintWhenMigrating = true
}

func dropForeignKeyConstraints(db *gorm.DB) error {
	var rows []foreignKeyConstraintRow
	if err := db.Raw(`
		SELECT TABLE_NAME, CONSTRAINT_NAME
		FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS
		WHERE TABLE_SCHEMA = DATABASE()
			AND CONSTRAINT_TYPE = 'FOREIGN KEY'
	`).Scan(&rows).Error; err != nil {
		return err
	}

	dropped := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if row.TableName == "" || row.ConstraintName == "" {
			continue
		}
		key := row.TableName + "." + row.ConstraintName
		if _, exists := dropped[key]; exists {
			continue
		}
		sql := fmt.Sprintf(
			"ALTER TABLE %s DROP FOREIGN KEY %s",
			quoteMySQLIdentifier(row.TableName),
			quoteMySQLIdentifier(row.ConstraintName),
		)
		if err := db.Exec(sql).Error; err != nil {
			return err
		}
		dropped[key] = struct{}{}
	}

	return nil
}

func quoteMySQLIdentifier(identifier string) string {
	return "`" + strings.ReplaceAll(identifier, "`", "``") + "`"
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
