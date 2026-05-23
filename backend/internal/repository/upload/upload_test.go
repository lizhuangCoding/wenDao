package upload

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

func TestUploadRepositoryListUnreferencedBefore_ExcludesDatabaseReferences(t *testing.T) {
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

	repo := NewUploadRepository(db)
	_, err = repo.ListUnreferencedBefore(time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC), 50)
	if err != nil {
		t.Fatalf("expected dry-run query to succeed, got %v", err)
	}

	joined := strings.Join(capture.statements, "\n")
	expectedFragments := []string{
		"FROM uploads AS uploads",
		"uploads.created_at < '2026-05-21 12:00:00'",
		"uploads.file_path LIKE '/uploads/%'",
		"articles.cover_image = uploads.file_path",
		"articles.content LIKE CONCAT('%', uploads.file_path, '%')",
		"articles.content_html LIKE CONCAT('%', uploads.file_path, '%')",
		"users.avatar_url = uploads.file_path",
		"ORDER BY uploads.created_at ASC LIMIT 50",
	}
	for _, fragment := range expectedFragments {
		if !strings.Contains(joined, fragment) {
			t.Fatalf("expected SQL fragment %q in statements:\n%s", fragment, joined)
		}
	}
}
