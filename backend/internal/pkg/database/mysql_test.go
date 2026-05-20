package database

import (
	"testing"

	"gorm.io/gorm/logger"
)

func TestNewGORMConfigDisablesForeignKeyConstraintsWhenMigrating(t *testing.T) {
	cfg := newGORMConfig(logger.Discard)

	if !cfg.DisableForeignKeyConstraintWhenMigrating {
		t.Fatal("expected GORM migration config to disable foreign key constraints")
	}
}
