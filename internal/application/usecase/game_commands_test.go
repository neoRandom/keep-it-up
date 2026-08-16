package usecase

import (
	"context"
	"database/sql"
	"testing"

	"keep-it-up/internal/infrastructure/database"

	_ "modernc.org/sqlite"
)

func newTestQueries(t *testing.T) *database.Queries {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if _, err = db.Exec("CREATE TABLE games (id INTEGER PRIMARY KEY, name TEXT NOT NULL)"); err != nil {
		t.Fatalf("create games table: %v", err)
	}

	return database.New(db)
}

func TestGameCommands_AddGameAndUpdateGame(t *testing.T) {
	ctx := context.Background()
	uc := NewGameCommands(newTestQueries(t))

	created, err := uc.AddGame(ctx, "Alpha")
	if err != nil {
		t.Fatalf("AddGame() returned error: %v", err)
	}
	if created.Name != "Alpha" {
		t.Fatalf("AddGame() created name = %q, want %q", created.Name, "Alpha")
	}

	if err := uc.UpdateGame(ctx, created.ID, "Bravo"); err != nil {
		t.Fatalf("UpdateGame() returned error: %v", err)
	}

	updated, err := uc.q.GetGame(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetGame() returned error after update: %v", err)
	}
	if updated.Name != "Bravo" {
		t.Fatalf("GetGame() name = %q, want %q", updated.Name, "Bravo")
	}
}

func TestGameCommands_RejectsInvalidGameNames(t *testing.T) {
	ctx := context.Background()
	uc := NewGameCommands(newTestQueries(t))

	if _, err := uc.AddGame(ctx, "A"); err == nil {
		t.Fatal("AddGame() accepted short name")
	}

	if _, err := uc.AddGame(ctx, "Bad-Name"); err == nil {
		t.Fatal("AddGame() accepted non-alphanumeric name")
	}

	if err := uc.UpdateGame(ctx, 1, "A"); err == nil {
		t.Fatal("UpdateGame() accepted short name")
	}
}

func TestGameCommands_DeleteGame(t *testing.T) {
	ctx := context.Background()
	queries := newTestQueries(t)
	uc := NewGameCommands(queries)

	created, err := uc.AddGame(ctx, "Gamma")
	if err != nil {
		t.Fatalf("AddGame() returned error: %v", err)
	}

	if err := uc.DeleteGame(ctx, created.ID); err != nil {
		t.Fatalf("DeleteGame() returned error: %v", err)
	}

	if _, err := queries.GetGame(ctx, created.ID); err == nil {
		t.Fatal("GetGame() found deleted game")
	}
}
