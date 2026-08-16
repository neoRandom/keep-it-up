package usecase

import (
	"context"
	"database/sql"
	"testing"

	"keep-it-up/internal/infrastructure/database"

	_ "modernc.org/sqlite"
)

func newPlayerTestQueries(t *testing.T) *database.Queries {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if _, err = db.Exec("CREATE TABLE players (id INTEGER PRIMARY KEY, name TEXT NOT NULL, username TEXT NOT NULL UNIQUE, hashed_password TEXT NOT NULL)"); err != nil {
		t.Fatalf("create players table: %v", err)
	}

	return database.New(db)
}

func TestPlayerManagement_AddPlayerAndUpdatePlayer(t *testing.T) {
	ctx := context.Background()
	uc := NewPlayerManagement(newPlayerTestQueries(t))

	created, err := uc.AddPlayer(ctx, "Alice", "alice", "secret123")
	if err != nil {
		t.Fatalf("AddPlayer() returned error: %v", err)
	}
	if created.Name != "Alice" {
		t.Fatalf("AddPlayer() created name = %q, want %q", created.Name, "Alice")
	}

	if err := uc.UpdatePlayerName(ctx, created.ID, "Alicia"); err != nil {
		t.Fatalf("UpdatePlayerName() returned error: %v", err)
	}
	if err := uc.UpdatePlayerPassword(ctx, created.ID, "newpass123"); err != nil {
		t.Fatalf("UpdatePlayerPassword() returned error: %v", err)
	}

	updated, err := uc.q.GetPlayer(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetPlayer() returned error after updates: %v", err)
	}
	if updated.Name != "Alicia" {
		t.Fatalf("GetPlayer() name = %q, want %q", updated.Name, "Alicia")
	}
	if updated.HashedPassword == "newpass123" {
		t.Fatal("GetPlayer() stored a plain-text password instead of a hash")
	}
}

func TestPlayerManagement_RejectsInvalidPlayerInput(t *testing.T) {
	ctx := context.Background()
	uc := NewPlayerManagement(newPlayerTestQueries(t))

	if _, err := uc.AddPlayer(ctx, "A", "alice", "secret123"); err == nil {
		t.Fatal("AddPlayer() accepted short name")
	}

	if _, err := uc.AddPlayer(ctx, "Alice", "al", "secret123"); err == nil {
		t.Fatal("AddPlayer() accepted short username")
	}

	if _, err := uc.AddPlayer(ctx, "Alice", "alice", "short"); err == nil {
		t.Fatal("AddPlayer() accepted short password")
	}

	if _, err := uc.AddPlayer(ctx, "Bad-Name", "bob", "secret123"); err == nil {
		t.Fatal("AddPlayer() accepted non-alphanumeric name")
	}

	if err := uc.UpdatePlayerName(ctx, 1, "A"); err == nil {
		t.Fatal("UpdatePlayerName() accepted short name")
	}

	if err := uc.UpdatePlayerPassword(ctx, 1, "short"); err == nil {
		t.Fatal("UpdatePlayerPassword() accepted short password")
	}
}

func TestPlayerManagement_DeletePlayer(t *testing.T) {
	ctx := context.Background()
	queries := newPlayerTestQueries(t)
	uc := NewPlayerManagement(queries)

	created, err := uc.AddPlayer(ctx, "Bob", "bob", "secret123")
	if err != nil {
		t.Fatalf("AddPlayer() returned error: %v", err)
	}

	if err := uc.DeletePlayer(ctx, created.ID); err != nil {
		t.Fatalf("DeletePlayer() returned error: %v", err)
	}

	if _, err := queries.GetPlayer(ctx, created.ID); err == nil {
		t.Fatal("GetPlayer() found deleted player")
	}
}
