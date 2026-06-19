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

func TestArticleRepositoryList_WithTagFilterQualifiesArticleColumns(t *testing.T) {
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
		TagID:    1,
		Page:     1,
		PageSize: 9,
	})
	if err != nil {
		t.Fatalf("expected tagged list dry-run to succeed, got %v", err)
	}

	joined := strings.Join(capture.statements, "\n")
	if !strings.Contains(joined, "JOIN article_tags ON article_tags.article_id = articles.id") {
		t.Fatalf("expected tag filter to join article_tags, statements:\n%s", joined)
	}
	for _, column := range articleListSelectColumns() {
		if !strings.HasPrefix(column, "articles.") {
			t.Fatalf("expected article list select column %q to be table-qualified", column)
		}
	}
	orderClause := articleListOrderClause(false)
	if !strings.Contains(orderClause, "articles.created_at") || strings.Contains(orderClause, " created_at") {
		t.Fatalf("expected article list order clause to qualify created_at, got %q", orderClause)
	}
}

func TestArticleRepositorySearch_UsesFullTextForArticleBodyKeyword(t *testing.T) {
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
	_, _, err = repo.Search(ArticleSearchFilter{
		Keyword:  "redis vector",
		Page:     1,
		PageSize: 10,
	})
	if err != nil && !strings.Contains(err.Error(), "dry run mode unsupported") {
		t.Fatalf("expected search dry-run to succeed, got %v", err)
	}

	joined := strings.Join(capture.statements, "\n")
	if !strings.Contains(joined, "MATCH(articles.title, articles.summary, articles.content) AGAINST") {
		t.Fatalf("expected keyword search to use fulltext MATCH, statements:\n%s", joined)
	}
	if strings.Contains(joined, "articles.content LIKE") {
		t.Fatalf("expected keyword search to avoid content LIKE scan, statements:\n%s", joined)
	}
	if !strings.Contains(joined, "categories.name LIKE") || !strings.Contains(joined, "tags.name LIKE") {
		t.Fatalf("expected keyword search to keep category/tag fallback matching, statements:\n%s", joined)
	}
}

func TestArticleRepositoryGetDueScheduledArticles_UsesApplicationTimeInsteadOfDatabaseNow(t *testing.T) {
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
	_, err = repo.GetDueScheduledArticles()
	if err != nil {
		t.Fatalf("expected due scheduled dry-run to succeed, got %v", err)
	}

	joined := strings.Join(capture.statements, "\n")
	if strings.Contains(joined, "NOW()") {
		t.Fatalf("expected due scheduled query to avoid database NOW(), statements:\n%s", joined)
	}
	if !strings.Contains(joined, "scheduled_publish_at <= '") {
		t.Fatalf("expected due scheduled query to compare against an application-time parameter, statements:\n%s", joined)
	}
}
