package comment

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

func newVoteDryRunRepository(t *testing.T) (*commentRepository, *sqlCaptureLogger) {
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

	return &commentRepository{db: db}, capture
}

func TestCommentRepositoryDecrementLikeDoesNotDropBelowZero(t *testing.T) {
	repo, capture := newVoteDryRunRepository(t)

	if err := repo.DecrementLike(7); err != nil {
		t.Fatalf("expected decrement like dry-run to succeed, got %v", err)
	}

	joined := strings.Join(capture.statements, "\n")
	if !strings.Contains(joined, "CASE WHEN like_count > 0 THEN like_count - 1 ELSE 0 END") {
		t.Fatalf("expected guarded like decrement SQL, got:\n%s", joined)
	}
}

func TestCommentRepositoryDecrementDislikeDoesNotDropBelowZero(t *testing.T) {
	repo, capture := newVoteDryRunRepository(t)

	if err := repo.DecrementDislike(7); err != nil {
		t.Fatalf("expected decrement dislike dry-run to succeed, got %v", err)
	}

	joined := strings.Join(capture.statements, "\n")
	if !strings.Contains(joined, "CASE WHEN dislike_count > 0 THEN dislike_count - 1 ELSE 0 END") {
		t.Fatalf("expected guarded dislike decrement SQL, got:\n%s", joined)
	}
}
