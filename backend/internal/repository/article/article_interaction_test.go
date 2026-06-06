package article

import (
	"database/sql"
	"strings"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"wenDao/internal/model"
)

func newArticleInteractionDryRunRepository(t *testing.T) (*articleRepository, *sqlCaptureLogger) {
	t.Helper()

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

	return &articleRepository{db: db}, capture
}

func TestArticleRepositoryListByInteraction_FiltersByUserTypeAndPublishedArticles(t *testing.T) {
	repo, capture := newArticleInteractionDryRunRepository(t)

	_, _, err := repo.ListByInteraction(9, model.ArticleInteractionTypeFavorite, ArticleFilter{Page: 2, PageSize: 12})
	if err != nil {
		t.Fatalf("expected interaction list dry-run to succeed, got %v", err)
	}

	joined := strings.Join(capture.statements, "\n")
	required := []string{
		"JOIN article_interactions",
		"article_interactions.user_id = 9",
		"article_interactions.interaction_type = 'favorite'",
		"articles.status = 'published'",
	}
	for _, expected := range required {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected SQL to contain %q, statements:\n%s", expected, joined)
		}
	}
}

func TestArticleRepositoryDelete_RemovesArticleInteractionsBeforeArticle(t *testing.T) {
	repo, capture := newArticleInteractionDryRunRepository(t)

	if err := repo.Delete(3); err != nil {
		t.Fatalf("expected delete dry-run to succeed, got %v", err)
	}

	joined := strings.Join(capture.statements, "\n")
	expected := "DELETE FROM `article_interactions` WHERE article_id = 3"
	if !strings.Contains(joined, expected) {
		t.Fatalf("expected article delete to remove interaction rows, statements:\n%s", joined)
	}
}
