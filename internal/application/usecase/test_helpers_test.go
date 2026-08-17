package usecase

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"keep-it-up/internal/infrastructure/database"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

func findMigrationsDir(t *testing.T) string {
	goose.SetLogger(goose.NopLogger())
	t.Helper()

	// Try multiple possible locations for migrations directory
	possiblePaths := []string{
		"database/migrations", // from project root
		filepath.Join("..", "..", "..", "database", "migrations"), // from test file location
	}

	// Also try computing from current working directory
	cwd, err := os.Getwd()
	if err == nil {
		possiblePaths = append(possiblePaths, filepath.Join(cwd, "database", "migrations"))
	}

	for _, path := range possiblePaths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path
		}
	}

	// If not found, fail with helpful message
	t.Fatalf("could not find migrations directory. Tried: %v", possiblePaths)
	return ""
}

func newTestDB(t *testing.T) *database.Queries {
	goose.SetLogger(goose.NopLogger())
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	// Ensure goose uses SQLite dialect
	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("set goose dialect: %v", err)
	}

	// Find and run Goose migrations to set up schema
	// This makes Goose migrations the single source of truth for schema
	migrationsDir := findMigrationsDir(t)

	// Run all migrations
	if err := goose.Up(db, migrationsDir); err != nil {
		t.Fatalf("run goose migrations: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	return database.New(db)
}
