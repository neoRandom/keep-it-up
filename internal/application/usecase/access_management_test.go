package usecase

import (
	"context"
	"strings"
	"testing"
)

func requireErrContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), want) {
		t.Errorf("expected error to contain %q, got %q", want, err.Error())
	}
}

func TestAccessManagement_GrantPlayerAccess(t *testing.T) {
	ctx := context.Background()

	t.Run("nil queries", func(t *testing.T) {
		if _, err := NewAccessManagement(nil); err == nil {
			t.Fatal("expected error for nil queries")
		}
	})

	for _, tt := range []struct {
		name     string
		gameID   int64
		playerID int64
		wantErr  string
	}{
		{"invalid game id: zero", 0, 1, "invalid game ID"},
		{"invalid game id: negative", -1, 1, "invalid game ID"},
		{"invalid player id: zero", 1, 0, "invalid player ID"},
		{"invalid player id: negative", 1, -1, "invalid player ID"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc := mustNewAccessManagement(t, newTestDB(t))
			err := uc.GrantPlayerAccess(ctx, tt.gameID, tt.playerID)
			requireErrContains(t, err, tt.wantErr)
		})
	}

	t.Run("valid pair succeeds", func(t *testing.T) {
		// SQLite disables foreign-key enforcement per-connection unless a
		// pragma turns it on, and nothing in the CLI-visible path does
		// that here, so this insert isn't expected to need pre-existing
		// game/player rows. If this fails against the real DB, FK
		// enforcement is on somewhere in the connection setup and
		// games/players need seeding first.
		uc := mustNewAccessManagement(t, newTestDB(t))
		if err := uc.GrantPlayerAccess(ctx, 1, 1); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("duplicate grant fails on the composite primary key", func(t *testing.T) {
		uc := mustNewAccessManagement(t, newTestDB(t))
		if err := uc.GrantPlayerAccess(ctx, 1, 1); err != nil {
			t.Fatalf("first grant: expected no error, got %v", err)
		}
		if err := uc.GrantPlayerAccess(ctx, 1, 1); err == nil {
			t.Fatal("second grant: expected a primary-key violation, got nil")
		}
	})
}

func TestAccessManagement_RevokePlayerAccess(t *testing.T) {
	ctx := context.Background()

	t.Run("nil queries", func(t *testing.T) {
		if _, err := NewAccessManagement(nil); err == nil {
			t.Fatal("expected error for nil queries")
		}
	})

	for _, tt := range []struct {
		name     string
		gameID   int64
		playerID int64
		wantErr  string
	}{
		{"invalid game id: zero", 0, 1, "invalid game ID"},
		{"invalid game id: negative", -1, 1, "invalid game ID"},
		{"invalid player id: zero", 1, 0, "invalid player ID"},
		{"invalid player id: negative", 1, -1, "invalid player ID"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc := mustNewAccessManagement(t, newTestDB(t))
			err := uc.RevokePlayerAccess(ctx, tt.gameID, tt.playerID)
			requireErrContains(t, err, tt.wantErr)
		})
	}

	t.Run("revoke after grant succeeds", func(t *testing.T) {
		uc := mustNewAccessManagement(t, newTestDB(t))
		if err := uc.GrantPlayerAccess(ctx, 1, 1); err != nil {
			t.Fatalf("setup grant: expected no error, got %v", err)
		}
		if err := uc.RevokePlayerAccess(ctx, 1, 1); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}
