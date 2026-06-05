package article

import (
	"database/sql"
	"strings"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestArticleRepositoryList_UsesLightweightColumnsAndAvoidsContentLikeSearch(t *testing.T) {
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
	_, _, err = repo.List(ArticleFilter{
		Status:   "published",
		Keyword:  "redis",
		Page:     1,
		PageSize: 9,
	})
	if err != nil {
		t.Fatalf("expected list dry-run to succeed, got %v", err)
	}

	joined := strings.Join(capture.statements, "\n")
	if strings.Contains(joined, "content LIKE") {
		t.Fatalf("expected keyword list search to avoid long content LIKE scan, statements:\n%s", joined)
	}
	if !strings.Contains(joined, "title LIKE") || !strings.Contains(joined, "summary LIKE") {
		t.Fatalf("expected keyword list search to keep title/summary predicates, statements:\n%s", joined)
	}
	if strings.Contains(joined, "`articles`.`content`") || strings.Contains(joined, "`content`") {
		t.Fatalf("expected public list query to omit article content column, statements:\n%s", joined)
	}
}
