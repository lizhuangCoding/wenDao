package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMigrationFilesSortsByNumericVersion(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{
		"000010_add_later.sql",
		"000001_initial_schema.sql",
		"notes.txt",
		"000002_add_users.sql",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("SELECT 1;"), 0o644); err != nil {
			t.Fatalf("write migration %s: %v", name, err)
		}
	}

	files, err := loadMigrationFiles(dir)
	if err != nil {
		t.Fatalf("expected migrations to load, got %v", err)
	}

	got := make([]string, 0, len(files))
	for _, file := range files {
		got = append(got, file.Version+":"+file.Name)
	}
	want := []string{
		"1:000001_initial_schema.sql",
		"2:000002_add_users.sql",
		"10:000010_add_later.sql",
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d files, got %d: %#v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected migration order %#v, got %#v", want, got)
		}
	}
}

func TestLoadMigrationFilesRejectsDuplicateNormalizedVersion(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{
		"1_initial.sql",
		"000001_duplicate.sql",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("SELECT 1;"), 0o644); err != nil {
			t.Fatalf("write migration %s: %v", name, err)
		}
	}

	if _, err := loadMigrationFiles(dir); err == nil {
		t.Fatal("expected duplicate normalized migration version to fail")
	}
}

func TestSplitSQLStatementsKeepsSemicolonsInsideQuotes(t *testing.T) {
	input := `
CREATE TABLE example (
  id BIGINT NOT NULL,
  note VARCHAR(255) DEFAULT 'hello;world',
  ` + "`key`" + ` VARCHAR(20)
);
INSERT INTO example (id, note, ` + "`key`" + `) VALUES (1, "a;b", 'c;d');
`

	statements := splitSQLStatements(input)
	want := []string{
		"CREATE TABLE example (\n  id BIGINT NOT NULL,\n  note VARCHAR(255) DEFAULT 'hello;world',\n  `key` VARCHAR(20)\n)",
		"INSERT INTO example (id, note, `key`) VALUES (1, \"a;b\", 'c;d')",
	}
	if len(statements) != len(want) {
		t.Fatalf("expected %d statements, got %d: %#v", len(want), len(statements), statements)
	}
	for i := range want {
		if statements[i] != want[i] {
			t.Fatalf("statement %d mismatch:\nwant: %q\n got: %q", i, want[i], statements[i])
		}
	}
}
