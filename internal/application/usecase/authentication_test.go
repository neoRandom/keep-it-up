package usecase

import (
	"context"
	"database/sql"
	"testing"

	"keep-it-up/internal/infrastructure/database"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

func newAuthTestQueries(t *testing.T) *database.Queries {
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

func TestAuthentication_VerifyPlayerPassword(t *testing.T) {
	auth := NewAuthentication(nil)

	if err := auth.VerifyPlayerPassword("secret123"); err != nil {
		t.Fatalf("VerifyPlayerPassword() rejected a valid password: %v", err)
	}

	if err := auth.VerifyPlayerPassword("short"); err == nil {
		t.Fatal("VerifyPlayerPassword() accepted a password shorter than 6 characters")
	}
}

func TestAuthentication_GeneratePasswordHash(t *testing.T) {
	auth := NewAuthentication(nil)

	hash, err := auth.GeneratePasswordHash("secret123")
	if err != nil {
		t.Fatalf("GeneratePasswordHash() returned error: %v", err)
	}
	if hash == "" {
		t.Fatal("GeneratePasswordHash() returned empty hash")
	}
	if hash == "secret123" {
		t.Fatal("GeneratePasswordHash() returned the raw password instead of a hash")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("secret123")); err != nil {
		t.Fatalf("bcrypt.CompareHashAndPassword() failed for generated hash: %v", err)
	}
}

func TestAuthentication_GeneratePasswordHashRejectsShortPassword(t *testing.T) {
	auth := NewAuthentication(nil)

	if _, err := auth.GeneratePasswordHash("short"); err == nil {
		t.Fatal("GeneratePasswordHash() accepted a password shorter than 6 characters")
	}
}

func TestAuthentication_CheckPlayerPasswordUsesPasswordRuleValidation(t *testing.T) {
	ctx := context.Background()
	queries := newAuthTestQueries(t)
	auth := NewAuthentication(queries)

	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword() returned error: %v", err)
	}

	_, err = queries.CreatePlayer(ctx, database.CreatePlayerParams{
		Name:           "Alice",
		Username:       "alice",
		HashedPassword: string(hash),
	})
	if err != nil {
		t.Fatalf("CreatePlayer() returned error: %v", err)
	}

	if _, err := auth.CheckPlayerPassword(ctx, "alice", "short"); err == nil {
		t.Fatal("CheckPlayerPassword() accepted a password that fails business-rule validation")
	}
}

func TestAuthentication_CheckPlayerPassword(t *testing.T) {
	ctx := context.Background()
	queries := newAuthTestQueries(t)
	auth := NewAuthentication(queries)

	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword() returned error: %v", err)
	}

	_, err = queries.CreatePlayer(ctx, database.CreatePlayerParams{
		Name:           "Alice",
		Username:       "alice",
		HashedPassword: string(hash),
	})
	if err != nil {
		t.Fatalf("CreatePlayer() returned error: %v", err)
	}

	ok, err := auth.CheckPlayerPassword(ctx, "alice", "secret123")
	if err != nil {
		t.Fatalf("CheckPlayerPassword() returned error: %v", err)
	}
	if !ok {
		t.Fatal("CheckPlayerPassword() returned false for the correct password")
	}
}

func TestAuthentication_CheckPlayerPasswordRejectsWrongPassword(t *testing.T) {
	ctx := context.Background()
	queries := newAuthTestQueries(t)
	auth := NewAuthentication(queries)

	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword() returned error: %v", err)
	}

	_, err = queries.CreatePlayer(ctx, database.CreatePlayerParams{
		Name:           "Alice",
		Username:       "alice",
		HashedPassword: string(hash),
	})
	if err != nil {
		t.Fatalf("CreatePlayer() returned error: %v", err)
	}

	ok, err := auth.CheckPlayerPassword(ctx, "alice", "wrongpass")
	if err != nil {
		t.Fatalf("CheckPlayerPassword() returned error for an incorrect password: %v", err)
	}
	if ok {
		t.Fatal("CheckPlayerPassword() returned true for the wrong password")
	}
}

func TestAuthentication_CheckPlayerPasswordRejectsUnknownUser(t *testing.T) {
	ctx := context.Background()
	auth := NewAuthentication(newAuthTestQueries(t))

	ok, err := auth.CheckPlayerPassword(ctx, "ghost", "secret123")
	if err == nil {
		t.Fatal("CheckPlayerPassword() did not return an error for an unknown username")
	}
	if ok {
		t.Fatal("CheckPlayerPassword() returned true for an unknown username")
	}
}

func TestAuthentication_CheckPlayerPasswordRejectsInvalidInput(t *testing.T) {
	ctx := context.Background()
	queries := newAuthTestQueries(t)
	auth := NewAuthentication(queries)

	if _, err := auth.CheckPlayerPassword(ctx, "a", "secret123"); err == nil {
		t.Fatal("CheckPlayerPassword() accepted a short username")
	}

	if _, err := auth.CheckPlayerPassword(ctx, "alice", "short"); err == nil {
		t.Fatal("CheckPlayerPassword() accepted a short password")
	}

	authNil := NewAuthentication(nil)
	if _, err := authNil.CheckPlayerPassword(ctx, "alice", "secret123"); err == nil {
		t.Fatal("CheckPlayerPassword() accepted nil database queries")
	}
}

func TestAuthentication_CheckPlayerPasswordRejectsMalformedHash(t *testing.T) {
	ctx := context.Background()
	queries := newAuthTestQueries(t)
	auth := NewAuthentication(queries)

	_, err := queries.CreatePlayer(ctx, database.CreatePlayerParams{
		Name:           "Alice",
		Username:       "alice",
		HashedPassword: "not-a-valid-bcrypt-hash",
	})
	if err != nil {
		t.Fatalf("CreatePlayer() returned error: %v", err)
	}

	ok, err := auth.CheckPlayerPassword(ctx, "alice", "secret123")
	if err == nil {
		t.Fatal("CheckPlayerPassword() did not return an error for a malformed stored hash")
	}
	if ok {
		t.Fatal("CheckPlayerPassword() returned true for a malformed stored hash")
	}
}
