package article

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"strings"
	"testing"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type sqlCaptureLogger struct {
	logger.Interface
	statements []string
}

func (l *sqlCaptureLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	sql, _ := fc()
	l.statements = append(l.statements, sql)
}

type noopConnector struct{}

func (noopConnector) Connect(ctx context.Context) (driver.Conn, error) {
	return noopConn{}, nil
}

func (noopConnector) Driver() driver.Driver {
	return noopDriver{}
}

type noopDriver struct{}

func (noopDriver) Open(name string) (driver.Conn, error) {
	return noopConn{}, nil
}

type noopConn struct{}

func (noopConn) Prepare(query string) (driver.Stmt, error) {
	return noopStmt{}, nil
}

func (noopConn) Close() error {
	return nil
}

func (noopConn) Begin() (driver.Tx, error) {
	return noopTx{}, nil
}

func (noopConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	return noopTx{}, nil
}

func (noopConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	return driver.RowsAffected(0), nil
}

func (noopConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return noopRows{}, nil
}

type noopStmt struct{}

func (noopStmt) Close() error {
	return nil
}

func (noopStmt) NumInput() int {
	return -1
}

func (noopStmt) Exec(args []driver.Value) (driver.Result, error) {
	return driver.RowsAffected(0), nil
}

func (noopStmt) Query(args []driver.Value) (driver.Rows, error) {
	return noopRows{}, nil
}

type noopTx struct{}

func (noopTx) Commit() error {
	return nil
}

func (noopTx) Rollback() error {
	return nil
}

type noopRows struct{}

func (noopRows) Columns() []string {
	return nil
}

func (noopRows) Close() error {
	return nil
}

func (noopRows) Next(dest []driver.Value) error {
	return io.EOF
}

func TestArticleRepositoryDelete_RemovesDependentRowsBeforeArticle(t *testing.T) {
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
	if err := repo.Delete(3); err != nil {
		t.Fatalf("expected delete dry-run to succeed, got %v", err)
	}

	joined := strings.Join(capture.statements, "\n")
	expectedOrder := []string{
		"DELETE FROM `comments` WHERE article_id = 3 AND parent_id IS NOT NULL",
		"DELETE FROM `comments` WHERE article_id = 3",
		"DELETE FROM `article_stats` WHERE article_id = 3",
		"UPDATE `knowledge_documents` SET `article_id`=NULL",
		"DELETE FROM `articles` WHERE `articles`.`id` = 3",
	}
	lastIndex := -1
	for _, expected := range expectedOrder {
		idx := strings.Index(joined[lastIndex+1:], expected)
		if idx < 0 {
			t.Fatalf("expected SQL %q in statements:\n%s", expected, joined)
		}
		idx += lastIndex + 1
		if idx <= lastIndex {
			t.Fatalf("expected SQL %q after previous dependent cleanup, statements:\n%s", expected, joined)
		}
		lastIndex = idx
	}
}

func TestArticleRepositoryDeleteBatch_RemovesDependentRowsWithInClauses(t *testing.T) {
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
	if _, err := repo.DeleteBatch([]int64{3, 5}); err != nil {
		t.Fatalf("expected batch delete dry-run to succeed, got %v", err)
	}

	joined := strings.Join(capture.statements, "\n")
	expected := []string{
		"SELECT * FROM `articles` WHERE id IN (3,5)",
		"DELETE FROM `comments` WHERE article_id IN (3,5) AND parent_id IS NOT NULL",
		"DELETE FROM `comments` WHERE article_id IN (3,5)",
		"DELETE FROM `article_stats` WHERE article_id IN (3,5)",
		"DELETE FROM `article_interactions` WHERE article_id IN (3,5)",
		"DELETE FROM `article_collections` WHERE article_id IN (3,5)",
		"DELETE FROM `article_tags` WHERE article_id IN (3,5)",
		"UPDATE `knowledge_documents` SET `article_id`=NULL",
		"DELETE FROM `articles` WHERE id IN (3,5)",
	}
	for _, sql := range expected {
		if !strings.Contains(joined, sql) {
			t.Fatalf("expected SQL %q in statements:\n%s", sql, joined)
		}
	}

	if strings.Contains(joined, "WHERE article_id = 3") || strings.Contains(joined, "WHERE article_id = 5") {
		t.Fatalf("expected batch delete to avoid per-article cleanup statements:\n%s", joined)
	}
}
