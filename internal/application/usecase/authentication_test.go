package usecase

import (
	"context"
	"testing"

	"keep-it-up/internal/infrastructure/database"

	"golang.org/x/crypto/bcrypt"
)

func TestAuthentication_CheckPlayerPasswordUsesPasswordRuleValidation(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	auth := mustNewAuthentication(t, queries, stubTG)

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
	queries := newTestDB(t)
	auth := mustNewAuthentication(t, queries, stubTG)

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

	player, err := auth.CheckPlayerPassword(ctx, "alice", "secret123")
	if err != nil {
		t.Fatalf("CheckPlayerPassword() returned error: %v", err)
	}
	if player.ID == 0 {
		t.Fatal("CheckPlayerPassword() returned false for the correct password")
	}
}

func TestAuthentication_CheckPlayerPasswordRejectsWrongPassword(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	auth := mustNewAuthentication(t, queries, stubTG)

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

	player, err := auth.CheckPlayerPassword(ctx, "alice", "wrongpass")
	if err != nil {
		t.Fatalf("CheckPlayerPassword() returned error for an incorrect password: %v", err)
	}
	if player.ID != 0 {
		t.Fatal("CheckPlayerPassword() returned true for the wrong password")
	}
}

func TestAuthentication_CheckPlayerPasswordRejectsUnknownUser(t *testing.T) {
	ctx := context.Background()
	auth := mustNewAuthentication(t, newTestDB(t), stubTG)

	player, err := auth.CheckPlayerPassword(ctx, "ghost", "secret123")
	if err == nil {
		t.Fatal("CheckPlayerPassword() did not return an error for an unknown username")
	}
	if player.ID != 0 {
		t.Fatal("CheckPlayerPassword() returned true for an unknown username")
	}
}

func TestAuthentication_CheckPlayerPasswordRejectsInvalidInput(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	auth := mustNewAuthentication(t, queries, stubTG)

	if _, err := auth.CheckPlayerPassword(ctx, "a", "secret123"); err == nil {
		t.Fatal("CheckPlayerPassword() accepted a short username")
	}

	if _, err := auth.CheckPlayerPassword(ctx, "alice", "short"); err == nil {
		t.Fatal("CheckPlayerPassword() accepted a short password")
	}

	if _, err := NewAuthentication(nil, nil); err == nil {
		t.Fatal("NewAuthentication() should return an error for nil queries")
	}
}

func TestAuthentication_CheckPlayerPasswordRejectsMalformedHash(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	auth := mustNewAuthentication(t, queries, stubTG)

	_, err := queries.CreatePlayer(ctx, database.CreatePlayerParams{
		Name:           "Alice",
		Username:       "alice",
		HashedPassword: "not-a-valid-bcrypt-hash",
	})
	if err != nil {
		t.Fatalf("CreatePlayer() returned error: %v", err)
	}

	player, err := auth.CheckPlayerPassword(ctx, "alice", "secret123")
	if err == nil {
		t.Fatal("CheckPlayerPassword() did not return an error for a malformed stored hash")
	}
	if player.ID != 0 {
		t.Fatal("CheckPlayerPassword() returned true for a malformed stored hash")
	}
}

func TestAuthentication_CheckPlayerPasswordWithCancelledContext(t *testing.T) {
	queries := newTestDB(t)
	auth := mustNewAuthentication(t, queries, stubTG)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := auth.CheckPlayerPassword(ctx, "alice", "secret123")
	if err == nil {
		t.Fatal("CheckPlayerPassword() should return error for cancelled context")
	}
}
