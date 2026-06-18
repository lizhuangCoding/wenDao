package database

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

const schemaMigrationsTable = "schema_migrations"

var migrationFilenamePattern = regexp.MustCompile(`^(\d+)[-_].+\.sql$`)

type VersionedMigrationOptions struct {
	Dir string
}

type migrationFile struct {
	Version string
	Name    string
	Path    string
}

func RunVersionedMigrations(db *gorm.DB, opts VersionedMigrationOptions) error {
	if db == nil {
		return fmt.Errorf("database is required")
	}
	dir := strings.TrimSpace(opts.Dir)
	if dir == "" {
		dir = "migrations"
	}

	files, err := loadMigrationFiles(dir)
	if err != nil {
		return err
	}
	if err := ensureSchemaMigrationsTable(db); err != nil {
		return err
	}

	applied, err := appliedMigrationVersions(db)
	if err != nil {
		return err
	}

	for _, file := range files {
		if _, ok := applied[file.Version]; ok {
			continue
		}
		if err := applyMigrationFile(db, file); err != nil {
			return err
		}
	}

	return nil
}

func loadMigrationFiles(dir string) ([]migrationFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations directory %q: %w", dir, err)
	}

	files := make([]migrationFile, 0, len(entries))
	seen := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		matches := migrationFilenamePattern.FindStringSubmatch(name)
		if matches == nil {
			continue
		}
		version := normalizeMigrationVersion(matches[1])
		if previous, exists := seen[version]; exists {
			return nil, fmt.Errorf("duplicate migration version %s in %s and %s", version, previous, name)
		}
		seen[version] = name
		files = append(files, migrationFile{
			Version: version,
			Name:    name,
			Path:    filepath.Join(dir, name),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		if len(files[i].Version) != len(files[j].Version) {
			return len(files[i].Version) < len(files[j].Version)
		}
		return files[i].Version < files[j].Version
	})

	return files, nil
}

func normalizeMigrationVersion(raw string) string {
	trimmed := strings.TrimLeft(raw, "0")
	if trimmed == "" {
		return "0"
	}
	return trimmed
}

func ensureSchemaMigrationsTable(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS schema_migrations (
	version VARCHAR(64) NOT NULL PRIMARY KEY,
	name VARCHAR(255) NOT NULL,
	applied_at DATETIME(3) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
`).Error
}

func appliedMigrationVersions(db *gorm.DB) (map[string]struct{}, error) {
	var rows []struct {
		Version string `gorm:"column:version"`
	}
	if err := db.Table(schemaMigrationsTable).Select("version").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("read applied migrations: %w", err)
	}

	applied := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		applied[normalizeMigrationVersion(row.Version)] = struct{}{}
	}
	return applied, nil
}

func applyMigrationFile(db *gorm.DB, file migrationFile) error {
	content, err := os.ReadFile(file.Path)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", file.Name, err)
	}
	statements := splitSQLStatements(string(content))
	if len(statements) == 0 {
		return fmt.Errorf("migration %s is empty", file.Name)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		for _, statement := range statements {
			if err := tx.Exec(statement).Error; err != nil {
				return fmt.Errorf("apply migration %s: %w", file.Name, err)
			}
		}
		if err := tx.Table(schemaMigrationsTable).Create(map[string]any{
			"version":    file.Version,
			"name":       file.Name,
			"applied_at": time.Now(),
		}).Error; err != nil {
			return fmt.Errorf("record migration %s: %w", file.Name, err)
		}
		return nil
	})
}

func splitSQLStatements(input string) []string {
	var statements []string
	var current strings.Builder
	var quote rune
	escaped := false

	for _, r := range input {
		current.WriteRune(r)

		if escaped {
			escaped = false
			continue
		}
		if r == '\\' && quote != 0 {
			escaped = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			}
			continue
		}
		if r == '\'' || r == '"' || r == '`' {
			quote = r
			continue
		}
		if r == ';' {
			statement := strings.TrimSpace(strings.TrimSuffix(current.String(), ";"))
			if statement != "" {
				statements = append(statements, statement)
			}
			current.Reset()
		}
	}

	if statement := strings.TrimSpace(current.String()); statement != "" {
		statements = append(statements, statement)
	}
	return statements
}
