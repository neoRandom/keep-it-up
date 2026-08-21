package usecase

import (
	"context"
	"testing"
)

func TestGameManagement_AddGame(t *testing.T) {
	ctx := context.Background()
	uc := NewGameManagement(newTestDB(t))

	created, err := uc.AddGame(ctx, "Alpha")
	if err != nil {
		t.Fatalf("AddGame() returned error: %v", err)
	}
	if created.Name != "Alpha" {
		t.Fatalf("AddGame() created name = %q, want %q", created.Name, "Alpha")
	}
}

func TestGameManagement_UpdateGame(t *testing.T) {
	ctx := context.Background()
	uc := NewGameManagement(newTestDB(t))

	created, err := uc.AddGame(ctx, "Alpha")
	if err != nil {
		t.Fatalf("setup: AddGame() returned error: %v", err)
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

func TestGameManagement_RejectsInvalidGameNames(t *testing.T) {
	ctx := context.Background()
	uc := NewGameManagement(newTestDB(t))

	if _, err := uc.AddGame(ctx, "A"); err == nil {
		t.Fatal("AddGame() accepted short name")
	}

	if _, err := uc.AddGame(ctx, "Bad-Name"); err != nil {
		t.Fatal("AddGame() rejected non-alphanumeric name")
	}

	if err := uc.UpdateGame(ctx, 1, "A"); err == nil {
		t.Fatal("UpdateGame() accepted short name")
	}

	if err := uc.UpdateGame(ctx, 1, "Bad-Name"); err != nil {
		t.Fatal("UpdateGame() rejected non-alphanumeric name")
	}
}

func TestGameManagement_RejectsNilDependency(t *testing.T) {
	ctx := context.Background()
	uc := NewGameManagement(nil)

	if _, err := uc.AddGame(ctx, "Alpha"); err == nil {
		t.Fatal("AddGame() accepted nil queries")
	}
	if err := uc.UpdateGame(ctx, 1, "Alpha"); err == nil {
		t.Fatal("UpdateGame() accepted nil queries")
	}
	if err := uc.DeleteGame(ctx, 1); err == nil {
		t.Fatal("DeleteGame() accepted nil queries")
	}
}

func TestGameManagement_RejectsInvalidIDs(t *testing.T) {
	ctx := context.Background()
	uc := NewGameManagement(newTestDB(t))

	for _, id := range []int64{0, -1} {
		if err := uc.UpdateGame(ctx, id, "Alpha"); err == nil {
			t.Fatalf("UpdateGame() accepted invalid id=%d", id)
		}
		if err := uc.DeleteGame(ctx, id); err == nil {
			t.Fatalf("DeleteGame() accepted invalid id=%d", id)
		}
	}
}

func TestGameManagement_DeleteGame(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	uc := NewGameManagement(queries)

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

func TestGameManagement_AddGameWithCancelledContext(t *testing.T) {
	uc := NewGameManagement(newTestDB(t))

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := uc.AddGame(ctx, "Alpha")
	if err == nil {
		t.Fatal("AddGame() should return error for cancelled context")
	}
}

func TestGameManagement_SuccessfulCRUDOperations(t *testing.T) {
	ctx := context.Background()
	uc := NewGameManagement(newTestDB(t))

	// Create
	created, err := uc.AddGame(ctx, "TestGame")
	if err != nil {
		t.Fatalf("AddGame() returned error: %v", err)
	}
	if created.Name != "TestGame" {
		t.Fatalf("AddGame() name = %q, want %q", created.Name, "TestGame")
	}
	gameID := created.ID

	// Read by fetching
	fetched, err := uc.q.GetGame(ctx, gameID)
	if err != nil {
		t.Fatalf("GetGame() returned error: %v", err)
	}
	if fetched.Name != "TestGame" {
		t.Fatalf("GetGame() returned wrong name: %q, want %q", fetched.Name, "TestGame")
	}

	// Update
	if err := uc.UpdateGame(ctx, gameID, "UpdatedGame"); err != nil {
		t.Fatalf("UpdateGame() returned error: %v", err)
	}

	// Verify update
	updated, err := uc.q.GetGame(ctx, gameID)
	if err != nil {
		t.Fatalf("GetGame() returned error after update: %v", err)
	}
	if updated.Name != "UpdatedGame" {
		t.Fatalf("GetGame() returned wrong name after update: %q, want %q", updated.Name, "UpdatedGame")
	}

	// Delete
	if err := uc.DeleteGame(ctx, gameID); err != nil {
		t.Fatalf("DeleteGame() returned error: %v", err)
	}

	// Verify deletion
	if _, err := uc.q.GetGame(ctx, gameID); err == nil {
		t.Fatal("GetGame() found game after DeleteGame()")
	}
}

func TestGameManagement_UpdateGameWithCancelledContext(t *testing.T) {
	uc := NewGameManagement(newTestDB(t))

	created, _ := uc.AddGame(context.Background(), "Alpha")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := uc.UpdateGame(ctx, created.ID, "Beta"); err == nil {
		t.Fatal("UpdateGame() should return error for cancelled context")
	}
}

func TestGameManagement_DeleteGameWithCancelledContext(t *testing.T) {
	uc := NewGameManagement(newTestDB(t))

	created, _ := uc.AddGame(context.Background(), "Gamma")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := uc.DeleteGame(ctx, created.ID); err == nil {
		t.Fatal("DeleteGame() should return error for cancelled context")
	}
}

func TestGameManagement_RejectsInvalidGameNameFormats(t *testing.T) {
	ctx := context.Background()
	uc := NewGameManagement(newTestDB(t))

	testCases := []struct {
		name  string
		valid bool
	}{
		{"AB", false},       // too short
		{"ABC", true},       // exactly 3 chars - minimum
		{"ValidGame", true}, // valid
		{"Game123", true},   // alphanumeric
		{"Game-Name", true}, // contains hyphen
		{"Game Name", true}, // contains space
		{"Game@123", true},  // contains special char
		{"123", true},       // digits only, 3 chars
		{"AB", false},       // 2 chars - below minimum
		{"A", false},        // 1 char - too short
		{"", false},         // empty
	}

	for _, tc := range testCases {
		_, err := uc.AddGame(ctx, tc.name)
		if tc.valid && err != nil {
			t.Fatalf("AddGame() should accept name %q: %v", tc.name, err)
		}
		if !tc.valid && err == nil {
			t.Fatalf("AddGame() should reject name %q", tc.name)
		}
	}
}

func TestGameManagement_GameNameBoundaryValidation(t *testing.T) {
	ctx := context.Background()
	uc := NewGameManagement(newTestDB(t))

	// Test at the boundary: exactly 3 characters
	if _, err := uc.AddGame(ctx, "ABC"); err != nil {
		t.Fatalf("AddGame() should accept exactly 3-char name: %v", err)
	}

	// Test below boundary: 2 characters
	if _, err := uc.AddGame(ctx, "AB"); err == nil {
		t.Fatal("AddGame() should reject 2-char name (below minimum of 3)")
	}

	// Test just above boundary: 4 characters
	if _, err := uc.AddGame(ctx, "ABCD"); err != nil {
		t.Fatalf("AddGame() should accept 4-char name: %v", err)
	}
}

func TestGameManagement_UpdateGameNameBoundaryValidation(t *testing.T) {
	ctx := context.Background()
	uc := NewGameManagement(newTestDB(t))

	created, _ := uc.AddGame(ctx, "InitialName")

	// Test at the boundary: exactly 3 characters
	if err := uc.UpdateGame(ctx, created.ID, "ABC"); err != nil {
		t.Fatalf("UpdateGame() should accept exactly 3-char name: %v", err)
	}

	// Test below boundary: 2 characters
	if err := uc.UpdateGame(ctx, created.ID, "AB"); err == nil {
		t.Fatal("UpdateGame() should reject 2-char name (below minimum of 3)")
	}
}

func TestGameManagement_RejectsNilQueriesDependencyInAllMethods(t *testing.T) {
	ctx := context.Background()

	// Create a GameManagement with nil queries
	uc := &GameManagement{q: nil}

	// All methods should reject nil queries
	if _, err := uc.AddGame(ctx, "Alpha"); err == nil {
		t.Fatal("AddGame() accepted nil queries")
	}

	if err := uc.UpdateGame(ctx, 1, "Beta"); err == nil {
		t.Fatal("UpdateGame() accepted nil queries")
	}

	if err := uc.DeleteGame(ctx, 1); err == nil {
		t.Fatal("DeleteGame() accepted nil queries")
	}
}

func TestGameManagement_RejectsInvalidIDsAcrossAllMethods(t *testing.T) {
	ctx := context.Background()
	uc := NewGameManagement(newTestDB(t))

	invalidIDs := []int64{-1, -100, 0}

	for _, id := range invalidIDs {
		if err := uc.UpdateGame(ctx, id, "Name"); err == nil {
			t.Fatalf("UpdateGame() should reject invalid id=%d", id)
		}

		if err := uc.DeleteGame(ctx, id); err == nil {
			t.Fatalf("DeleteGame() should reject invalid id=%d", id)
		}
	}
}

func TestGameManagement_AddGameEmptyName(t *testing.T) {
	ctx := context.Background()
	uc := NewGameManagement(newTestDB(t))

	if _, err := uc.AddGame(ctx, ""); err == nil {
		t.Fatal("AddGame() should reject empty name")
	}
}

func TestGameManagement_SuccessfulGameOperationSequence(t *testing.T) {
	ctx := context.Background()
	uc := NewGameManagement(newTestDB(t))

	// Create multiple games
	games := make([]int64, 3)
	gameNames := []string{"FirstGame", "SecondGame", "ThirdGame"}

	for i, name := range gameNames {
		game, err := uc.AddGame(ctx, name)
		if err != nil {
			t.Fatalf("AddGame(%q) returned error: %v", name, err)
		}
		games[i] = game.ID
	}

	// Update all games
	updatedNames := []string{"FirstUpdated", "SecondUpdated", "ThirdUpdated"}
	for i, name := range updatedNames {
		if err := uc.UpdateGame(ctx, games[i], name); err != nil {
			t.Fatalf("UpdateGame(%d, %q) returned error: %v", games[i], name, err)
		}
	}

	// Verify updates
	for i, expectedName := range updatedNames {
		game, err := uc.q.GetGame(ctx, games[i])
		if err != nil {
			t.Fatalf("GetGame(%d) returned error: %v", games[i], err)
		}
		if game.Name != expectedName {
			t.Fatalf("GetGame(%d) returned wrong name: %q, want %q", games[i], game.Name, expectedName)
		}
	}

	// Delete all games
	for _, id := range games {
		if err := uc.DeleteGame(ctx, id); err != nil {
			t.Fatalf("DeleteGame(%d) returned error: %v", id, err)
		}
	}

	// Verify deletions
	for _, id := range games {
		_, err := uc.q.GetGame(ctx, id)
		if err == nil {
			t.Fatalf("GetGame(%d) should fail after DeleteGame()", id)
		}
	}
}
