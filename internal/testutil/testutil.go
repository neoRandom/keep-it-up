// Package testutil provides shared test helpers used across packages (schema
// setup and access-scoped game readback) so they are defined once rather than
// duplicated in each _test package.
package testutil

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

// FindMigrationsDir locates the Goose migrations directory from wherever the
// test runs. It fails the test when the directory cannot be found.
func FindMigrationsDir(t *testing.T) string {
	goose.SetLogger(goose.NopLogger())
	t.Helper()

	possiblePaths := []string{
		"database/migrations",
		filepath.Join("..", "..", "..", "database", "migrations"),
	}
	if cwd, err := os.Getwd(); err == nil {
		possiblePaths = append(possiblePaths, filepath.Join(cwd, "database", "migrations"))
	}

	for _, path := range possiblePaths {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path
		}
	}
	t.Fatalf("could not find migrations directory. Tried: %v", possiblePaths)
	return ""
}

// NewTestDB opens an in-memory SQLite DB, applies the Goose migrations (the
// schema source of truth), and returns the SQLC queries.
func NewTestDB(t *testing.T) *database.Queries {
	goose.SetLogger(goose.NopLogger())
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("set goose dialect: %v", err)
	}
	if err := goose.Up(db, FindMigrationsDir(t)); err != nil {
		t.Fatalf("run goose migrations: %v", err)
	}
	return database.New(db)
}

// ReadGameNameByAccess reads a game name through the production fetching path
// (GrantPlayerAccess + ListPlayerGames), reporting whether the game is visible
// to an authorized player.
//
// NOTE (Phase C): there is no direct game-by-ID query, so this uses the
// access-scoped readback rather than a minimal read. A domain-level Game fetch
// may eliminate this workaround.
func ReadGameNameByAccess(t *testing.T, ctx context.Context, q *database.Queries, gameID int64) (string, bool) {
	t.Helper()
	player, err := q.CreatePlayer(ctx, database.CreatePlayerParams{
		Name:           fmt.Sprintf("reader_%d", time.Now().UnixNano()),
		Username:       fmt.Sprintf("reader_%d", time.Now().UnixNano()),
		HashedPassword: "hash",
	})
	if err != nil {
		t.Fatalf("ReadGameNameByAccess: CreatePlayer: %v", err)
	}
	if _, err := q.GrantPlayerAccess(ctx, database.GrantPlayerAccessParams{GameID: gameID, PlayerID: player.ID}); err != nil {
		t.Fatalf("ReadGameNameByAccess: GrantPlayerAccess: %v", err)
	}
	games, err := q.ListPlayerGames(ctx, player.ID)
	if err != nil {
		t.Fatalf("ReadGameNameByAccess: ListPlayerGames: %v", err)
	}
	for _, g := range games {
		if g.ID == gameID {
			return g.Name, true
		}
	}
	return "", false
}
