package article

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestArticleRepositoryUpdatePopularityScores_UsesSingleBulkUpdate(t *testing.T) {
	capture := &sqlCaptureLogger{Interface: logger.Discard}
	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sql.OpenDB(noopConnector{}),
		SkipInitializeWithVersion: true,
	}), &gorm.Config{
		DryRun: true,
		Logger: capture,
	})
	if err != nil {
		t.Fatalf("expected dry-run database to open, got %v", err)
	}

	repo := NewArticleRepository(db)
	if err := repo.UpdatePopularityScores(time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("expected popularity update dry-run to succeed, got %v", err)
	}

	joined := strings.Join(capture.statements, "\n")
	if got := strings.Count(joined, "UPDATE articles"); got != 1 {
		t.Fatalf("expected exactly one bulk article update, got %d statements:\n%s", got, joined)
	}
	for _, expected := range []string{
		"SET popularity =",
		"view_count",
		"comment_count",
		"like_count",
		"POW(",
		"WHERE status = 'published'",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected SQL to contain %q, statements:\n%s", expected, joined)
		}
	}
}
