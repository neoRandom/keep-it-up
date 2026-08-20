package usecase

import (
	"context"
	"testing"
)

func TestDataFetching_ListPlayerGames(t *testing.T) {
	ctx := context.Background()

	t.Run("nil queries", func(t *testing.T) {
		uc := NewDataFetching(nil)
		_, err := uc.ListPlayerGames(ctx, 1)
		requireErrContains(t, err, "not initialized")
	})

	for _, tt := range []struct {
		name     string
		playerID int64
		wantErr  string
	}{
		{"invalid player id: zero", 0, "Invalid player ID"},
		{"invalid player id: negative", -1, "Invalid player ID"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewDataFetching(newTestDB(t))
			_, err := uc.ListPlayerGames(ctx, tt.playerID)
			requireErrContains(t, err, tt.wantErr)
		})
	}

	t.Run("player with no games returns empty, no error", func(t *testing.T) {
		uc := NewDataFetching(newTestDB(t))
		games, err := uc.ListPlayerGames(ctx, 1)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(games) != 0 {
			t.Errorf("expected no games, got %v", games)
		}
	})
}

func TestDataFetching_GetSharedData(t *testing.T) {
	ctx := context.Background()

	t.Run("nil queries", func(t *testing.T) {
		uc := NewDataFetching(nil)
		_, err := uc.GetSharedData(ctx, 1)
		requireErrContains(t, err, "not initialized")
	})

	for _, tt := range []struct {
		name    string
		gameID  int64
		wantErr string
	}{
		{"invalid game id: zero", 0, "Invalid game ID"},
		{"invalid game id: negative", -1, "Invalid game ID"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewDataFetching(newTestDB(t))
			_, err := uc.GetSharedData(ctx, tt.gameID)
			requireErrContains(t, err, tt.wantErr)
		})
	}

	// Success path not covered: it calls service.BuildSharedData after
	// ListInteractionsForReplay, and that function's contract (does it
	// error on a game with zero interactions? what shape does it return?)
	// isn't visible from access_management.go/data_fetching.go. Send the
	// service package if you want this closed out too.
}

func TestDataFetching_ListInteractions(t *testing.T) {
	ctx := context.Background()

	t.Run("nil queries", func(t *testing.T) {
		uc := NewDataFetching(nil)
		_, err := uc.ListInteractions(ctx, 1, 10)
		requireErrContains(t, err, "not initialized")
	})

	for _, tt := range []struct {
		name    string
		gameID  int64
		limit   int64
		wantErr string
	}{
		{"invalid game id: zero", 0, 10, "Invalid game ID"},
		{"invalid game id: negative", -1, 10, "Invalid game ID"},
		{"negative limit", 1, -1, "query limit cannot be less than 0"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewDataFetching(newTestDB(t))
			_, err := uc.ListInteractions(ctx, tt.gameID, tt.limit)
			requireErrContains(t, err, tt.wantErr)
		})
	}

	t.Run("game with no interactions returns empty, no error", func(t *testing.T) {
		uc := NewDataFetching(newTestDB(t))
		interactions, err := uc.ListInteractions(ctx, 1, 10)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(interactions) != 0 {
			t.Errorf("expected no interactions, got %v", interactions)
		}
	})

	t.Run("limit zero is a valid boundary", func(t *testing.T) {
		uc := NewDataFetching(newTestDB(t))
		if _, err := uc.ListInteractions(ctx, 1, 0); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}
