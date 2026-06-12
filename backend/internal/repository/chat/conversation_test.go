package chat

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"strings"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"wenDao/internal/model"
)

type noopConnector struct{}

func (noopConnector) Connect(context.Context) (driver.Conn, error) {
	return noopConn{}, nil
}

func (noopConnector) Driver() driver.Driver {
	return noopDriver{}
}

type noopDriver struct{}

func (noopDriver) Open(string) (driver.Conn, error) {
	return noopConn{}, nil
}

type noopConn struct{}

func (noopConn) Prepare(string) (driver.Stmt, error) {
	return noopStmt{}, nil
}

func (noopConn) Close() error {
	return nil
}

func (noopConn) Begin() (driver.Tx, error) {
	return noopTx{}, nil
}

type noopStmt struct{}

func (noopStmt) Close() error {
	return nil
}

func (noopStmt) NumInput() int {
	return -1
}

func (noopStmt) Exec([]driver.Value) (driver.Result, error) {
	return driver.RowsAffected(0), nil
}

func (noopStmt) Query([]driver.Value) (driver.Rows, error) {
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

func (noopRows) Next([]driver.Value) error {
	return io.EOF
}

func newDryRunConversationRepository(t *testing.T) *conversationRepository {
	t.Helper()
	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sql.OpenDB(noopConnector{}),
		SkipInitializeWithVersion: true,
	}), &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("expected dry-run database to open, got %v", err)
	}
	return NewConversationRepository(db).(*conversationRepository)
}

func TestConversationRepositoryCreate_OmitsEmptyShareToken(t *testing.T) {
	repo := newDryRunConversationRepository(t)

	tx := repo.createQuery(&model.Conversation{UserID: 2, Title: "新会话"})
	sql := tx.Statement.SQL.String()
	if strings.Contains(sql, "share_token") {
		t.Fatalf("expected create SQL to omit empty share_token unique column, got %s", sql)
	}
}

func TestConversationRepositoryUpdate_OmitsEmptyShareToken(t *testing.T) {
	repo := newDryRunConversationRepository(t)

	tx := repo.updateQuery(&model.Conversation{ID: 7, UserID: 2, Title: "帮我调研一下马斯克"})
	sql := tx.Statement.SQL.String()
	if strings.Contains(sql, "share_token") {
		t.Fatalf("expected update SQL to omit empty share_token unique column, got %s", sql)
	}
	if !strings.Contains(sql, "`title`=?") {
		t.Fatalf("expected update SQL to include title, got %s", sql)
	}
}

func TestConversationRepositoryUpdateShare_ClearsTokenWithNull(t *testing.T) {
	repo := newDryRunConversationRepository(t)

	tx := repo.updateShareQuery(7, false, "")
	sql := tx.Statement.SQL.String()
	if !strings.Contains(sql, "`share_token`=?") {
		t.Fatalf("expected update SQL to set share_token, got %s", sql)
	}
	if len(tx.Statement.Vars) < 2 || tx.Statement.Vars[1] != nil {
		t.Fatalf("expected share_token update value to be nil, got %#v in vars %#v", tx.Statement.Vars[1], tx.Statement.Vars)
	}
}
