package database

import (
	"testing"

	"gorm.io/gorm"

	"wenDao/internal/model"
)

func TestDisableForeignKeyConstraintsWhenMigrating(t *testing.T) {
	db := &gorm.DB{Config: &gorm.Config{}}

	disableForeignKeyConstraintsWhenMigrating(db)

	if !db.DisableForeignKeyConstraintWhenMigrating {
		t.Fatal("expected AutoMigrate path to disable foreign key constraints")
	}
}

func TestQuoteMySQLIdentifierEscapesBackticks(t *testing.T) {
	got := quoteMySQLIdentifier("bad`name")
	want := "`bad``name`"

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestAutoMigrateModelsIncludesNotification(t *testing.T) {
	for _, migrationModel := range autoMigrateModels() {
		if _, ok := migrationModel.(*model.Notification); ok {
			return
		}
	}

	t.Fatal("expected AutoMigrate model list to include Notification")
}
