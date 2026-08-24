package usecase

import (
	"context"
	"testing"

	"keep-it-up/internal/testutil"
)

func TestGameManagement_CRUDSequence(t *testing.T) {
	ctx := context.Background()
	uc := mustNewGameManagement(t, newTestDB(t))

	// Create
	created, err := uc.AddGame(ctx, "TestGame")
	if err != nil {
		t.Fatalf("AddGame() returned error: %v", err)
	}
	if created.Name != "TestGame" {
		t.Fatalf("AddGame() name = %q, want %q", created.Name, "TestGame")
	}
	gameID := created.ID

	// Read via the access-scoped fetching path
	name, ok := testutil.ReadGameNameByAccess(t, ctx, uc.q, gameID)
	if !ok {
		t.Fatal("game not readable after create")
	}
	if name != "TestGame" {
		t.Fatalf("game name = %q, want %q", name, "TestGame")
	}

	// Update
	if err := uc.UpdateGame(ctx, gameID, "UpdatedGame"); err != nil {
		t.Fatalf("UpdateGame() returned error: %v", err)
	}
	name, ok = testutil.ReadGameNameByAccess(t, ctx, uc.q, gameID)
	if !ok {
		t.Fatal("game not readable after update")
	}
	if name != "UpdatedGame" {
		t.Fatalf("game name after update = %q, want %q", name, "UpdatedGame")
	}

	// Delete
	if err := uc.DeleteGame(ctx, gameID); err != nil {
		t.Fatalf("DeleteGame() returned error: %v", err)
	}
	if _, ok := testutil.ReadGameNameByAccess(t, ctx, uc.q, gameID); ok {
		t.Fatal("testutil.ReadGameNameByAccess() found game after DeleteGame()")
	}
}

func TestGameManagement_RejectsNilQueries(t *testing.T) {
	if _, err := NewGameManagement(nil); err == nil {
		t.Fatal("NewGameManagement() should return an error for nil queries")
	}
}

func TestGameManagement_RejectsInvalidIDs(t *testing.T) {
	ctx := context.Background()
	uc := mustNewGameManagement(t, newTestDB(t))

	for _, id := range []int64{-1, -100, 0} {
		if err := uc.UpdateGame(ctx, id, "Name"); err == nil {
			t.Fatalf("UpdateGame() accepted invalid id=%d", id)
		}
		if err := uc.DeleteGame(ctx, id); err == nil {
			t.Fatalf("DeleteGame() accepted invalid id=%d", id)
		}
	}
}

func TestGameManagement_RejectsInvalidNames(t *testing.T) {
	ctx := context.Background()
	uc := mustNewGameManagement(t, newTestDB(t))

	// AddGame name rule: minimum 3 characters; non-alphanumeric is allowed.
	for _, tc := range []struct {
		name  string
		valid bool
	}{
		{"AB", false},
		{"ABC", true},
		{"A", false},
		{"", false},
		{"ValidGame", true},
		{"Game-Name", true},
		{"Game Name", true},
		{"Game@123", true},
		{"123", true},
	} {
		_, err := uc.AddGame(ctx, tc.name)
		if tc.valid && err != nil {
			t.Fatalf("AddGame() should accept name %q: %v", tc.name, err)
		}
		if !tc.valid && err == nil {
			t.Fatalf("AddGame() should reject name %q", tc.name)
		}
	}

	// UpdateGame applies the same minimum-length rule.
	created, err := uc.AddGame(ctx, "InitialName")
	if err != nil {
		t.Fatalf("setup AddGame: %v", err)
	}
	if err := uc.UpdateGame(ctx, created.ID, "ABC"); err != nil {
		t.Fatalf("UpdateGame() should accept 3-char name: %v", err)
	}
	if err := uc.UpdateGame(ctx, created.ID, "AB"); err == nil {
		t.Fatal("UpdateGame() accepted 2-char name (below minimum)")
	}
	if err := uc.UpdateGame(ctx, created.ID, ""); err == nil {
		t.Fatal("UpdateGame() accepted empty name")
	}
}

func TestGameManagement_CancelledContext(t *testing.T) {

	tests := []struct {
		name string
		run  func(ctx context.Context, uc *GameManagement, id int64) error
	}{
		{"AddGame", func(ctx context.Context, uc *GameManagement, _ int64) error {
			_, err := uc.AddGame(ctx, "Alpha")
			return err
		}},
		{"UpdateGame", func(ctx context.Context, uc *GameManagement, id int64) error {
			return uc.UpdateGame(ctx, id, "Beta")
		}},
		{"DeleteGame", func(ctx context.Context, uc *GameManagement, id int64) error {
			return uc.DeleteGame(ctx, id)
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			uc := mustNewGameManagement(t, newTestDB(t))
			created, err := uc.AddGame(context.Background(), "Seed")
			if err != nil {
				t.Fatalf("setup AddGame: %v", err)
			}

			cctx, cancel := context.WithCancel(context.Background())
			cancel()

			if err := tc.run(cctx, uc, created.ID); err == nil {
				t.Fatalf("%s() should return error for cancelled context", tc.name)
			}
		})
	}
}
