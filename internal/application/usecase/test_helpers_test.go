package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

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

// readGameNameByAccess reads a game's name through the production fetching
// path (GrantPlayerAccess + ListPlayerGames), reporting whether the game is
// visible to an authorized player. Using the access-scoped readback keeps the
// test suite exercising real production queries rather than a now-removed
// GetGame helper.
func readGameNameByAccess(t *testing.T, ctx context.Context, q *database.Queries, gameID int64) (string, bool) {
	t.Helper()
	player, err := q.CreatePlayer(ctx, database.CreatePlayerParams{
		Name:           fmt.Sprintf("reader_%d", time.Now().UnixNano()),
		Username:       fmt.Sprintf("reader_%d", time.Now().UnixNano()),
		HashedPassword: "hash",
	})
	if err != nil {
		t.Fatalf("readGameNameByAccess: CreatePlayer: %v", err)
	}
	if _, err := q.GrantPlayerAccess(ctx, database.GrantPlayerAccessParams{GameID: gameID, PlayerID: player.ID}); err != nil {
		t.Fatalf("readGameNameByAccess: GrantPlayerAccess: %v", err)
	}
	games, err := q.ListPlayerGames(ctx, player.ID)
	if err != nil {
		t.Fatalf("readGameNameByAccess: ListPlayerGames: %v", err)
	}
	for _, g := range games {
		if g.ID == gameID {
			return g.Name, true
		}
	}
	return "", false
}
