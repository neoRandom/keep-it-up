package usecase

import (
	"context"
	"testing"
)

func TestDataFetching_ListPlayerGames(t *testing.T) {
	ctx := context.Background()

	t.Run("nil queries", func(t *testing.T) {
		uc := NewDataFetching(nil, newFixedClock())
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
			uc := NewDataFetching(newTestDB(t), newFixedClock())
			_, err := uc.ListPlayerGames(ctx, tt.playerID)
			requireErrContains(t, err, tt.wantErr)
		})
	}

	t.Run("player with no games returns empty, no error", func(t *testing.T) {
		uc := NewDataFetching(newTestDB(t), newFixedClock())
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
		uc := NewDataFetching(nil, newFixedClock())
		_, err := uc.GetSharedData(ctx, 1)
		requireErrContains(t, err, "not initialized")
	})

	t.Run("nil time provider", func(t *testing.T) {
		uc := NewDataFetching(newTestDB(t), nil)
		_, err := uc.GetSharedData(ctx, 1)
		requireErrContains(t, err, "time provider is not initialized")
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
			uc := NewDataFetching(newTestDB(t), newFixedClock())
			_, err := uc.GetSharedData(ctx, tt.gameID)
			requireErrContains(t, err, tt.wantErr)
		})
	}

	t.Run("game with no interactions returns not-started shared data", func(t *testing.T) {
		uc := NewDataFetching(newTestDB(t), newFixedClock())
		shared, err := uc.GetSharedData(ctx, 1)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if shared.GameID != 1 {
			t.Errorf("GameID = %d, want 1", shared.GameID)
		}
		if shared.Status != "not_started" {
			t.Errorf("Status = %q, want not_started", shared.Status)
		}
	})

	t.Run("valid flag is computed when a deadline is present", func(t *testing.T) {
		q := newTestDB(t)
		uc := NewDataFetching(q, newFixedClock())

		// A save at fixed-clock time 12:00:00 with a 60s duration sets the
		// deadline to 12:01:00, which is in the future relative to "now", so
		// valid must be true — proving ComputeValid runs inside GetSharedData.
		commands := NewGameCommands(q, newFixedClock())
		if err := commands.SaveGame(ctx, 5, 1, 60); err != nil {
			t.Fatalf("setup SaveGame: %v", err)
		}

		shared, err := uc.GetSharedData(ctx, 5)
		if err != nil {
			t.Fatalf("GetSharedData: %v", err)
		}
		if shared.Valid == nil {
			t.Fatal("expected non-nil Valid computed by GetSharedData")
		}
		if !*shared.Valid {
			t.Errorf("Valid = false, want true (deadline is in the future)")
		}
	})
}

func TestDataFetching_FirstInteraction(t *testing.T) {
	ctx := context.Background()

	t.Run("nil queries", func(t *testing.T) {
		uc := NewDataFetching(nil, newFixedClock())
		_, err := uc.FirstInteraction(ctx, 1)
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
			uc := NewDataFetching(newTestDB(t), newFixedClock())
			_, err := uc.FirstInteraction(ctx, tt.gameID)
			requireErrContains(t, err, tt.wantErr)
		})
	}

	t.Run("game with no interactions returns nil, no error", func(t *testing.T) {
		uc := NewDataFetching(newTestDB(t), newFixedClock())
		interaction, err := uc.FirstInteraction(ctx, 1)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if interaction != nil {
			t.Errorf("expected nil interaction, got %+v", interaction)
		}
	})

	t.Run("returns the earliest interaction of the game", func(t *testing.T) {
		q := newTestDB(t)
		commands := NewGameCommands(q, newFixedClock())

		// save (12:00:00), pause (12:00:01), resume (12:00:02), save (12:00:03).
		if err := commands.SaveGame(ctx, 5, 1, 60); err != nil {
			t.Fatalf("setup save 1: %v", err)
		}
		if err := commands.PauseGame(ctx, 5, 1); err != nil {
			t.Fatalf("setup pause: %v", err)
		}
		if err := commands.ResumeGame(ctx, 5, 1); err != nil {
			t.Fatalf("setup resume: %v", err)
		}
		if err := commands.SaveGame(ctx, 5, 1, 60); err != nil {
			t.Fatalf("setup save 2: %v", err)
		}

		uc := NewDataFetching(q, newFixedClock())
		interaction, err := uc.FirstInteraction(ctx, 5)
		if err != nil {
			t.Fatalf("FirstInteraction: %v", err)
		}
		if interaction == nil {
			t.Fatal("expected non-nil interaction")
		}
		if interaction.Action != "saved" || interaction.OccurredAt != "2026-08-22T12:00:00Z" {
			t.Errorf("first interaction = %+v, want the initial save", interaction)
		}
	})
}

func TestDataFetching_LastInteraction(t *testing.T) {
	ctx := context.Background()

	t.Run("nil queries", func(t *testing.T) {
		uc := NewDataFetching(nil, newFixedClock())
		_, err := uc.LastInteraction(ctx, 1)
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
			uc := NewDataFetching(newTestDB(t), newFixedClock())
			_, err := uc.LastInteraction(ctx, tt.gameID)
			requireErrContains(t, err, tt.wantErr)
		})
	}

	t.Run("game with no interactions returns nil, no error", func(t *testing.T) {
		uc := NewDataFetching(newTestDB(t), newFixedClock())
		interaction, err := uc.LastInteraction(ctx, 1)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if interaction != nil {
			t.Errorf("expected nil interaction, got %+v", interaction)
		}
	})

	t.Run("returns the latest interaction of the game", func(t *testing.T) {
		q := newTestDB(t)
		commands := NewGameCommands(q, newFixedClock())

		// save (12:00:00), pause (12:00:01), resume (12:00:02), save (12:00:03).
		if err := commands.SaveGame(ctx, 5, 1, 60); err != nil {
			t.Fatalf("setup save 1: %v", err)
		}
		if err := commands.PauseGame(ctx, 5, 1); err != nil {
			t.Fatalf("setup pause: %v", err)
		}
		if err := commands.ResumeGame(ctx, 5, 1); err != nil {
			t.Fatalf("setup resume: %v", err)
		}
		if err := commands.SaveGame(ctx, 5, 1, 60); err != nil {
			t.Fatalf("setup save 2: %v", err)
		}

		uc := NewDataFetching(q, newFixedClock())
		interaction, err := uc.LastInteraction(ctx, 5)
		if err != nil {
			t.Fatalf("LastInteraction: %v", err)
		}
		if interaction == nil {
			t.Fatal("expected non-nil interaction")
		}
		if interaction.Action != "saved" || interaction.OccurredAt != "2026-08-22T12:00:03Z" {
			t.Errorf("last interaction = %+v, want the final save", interaction)
		}
	})
}

func TestDataFetching_ListPlayerInteractions(t *testing.T) {
	ctx := context.Background()

	t.Run("nil queries", func(t *testing.T) {
		uc := NewDataFetching(nil, newFixedClock())
		_, err := uc.ListPlayerInteractions(ctx, 1)
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
			uc := NewDataFetching(newTestDB(t), newFixedClock())
			_, err := uc.ListPlayerInteractions(ctx, tt.playerID)
			requireErrContains(t, err, tt.wantErr)
		})
	}

	t.Run("player with no interactions returns empty, no error", func(t *testing.T) {
		uc := NewDataFetching(newTestDB(t), newFixedClock())
		interactions, err := uc.ListPlayerInteractions(ctx, 1)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(interactions) != 0 {
			t.Errorf("expected no interactions, got %v", interactions)
		}
	})

	t.Run("returns only the given player's interactions, newest first", func(t *testing.T) {
		q := newTestDB(t)
		commands := NewGameCommands(q, newFixedClock())

		// Player 1 performs a save on game 1, then player 2 on game 2, then
		// player 1 again on game 3. The fixed clock advances one second per
		// call, so timestamps are strictly increasing.
		if err := commands.SaveGame(ctx, 1, 1, 60); err != nil {
			t.Fatalf("setup save 1: %v", err)
		}
		if err := commands.SaveGame(ctx, 2, 2, 60); err != nil {
			t.Fatalf("setup save 2: %v", err)
		}
		if err := commands.SaveGame(ctx, 3, 1, 60); err != nil {
			t.Fatalf("setup save 3: %v", err)
		}

		uc := NewDataFetching(q, newFixedClock())
		interactions, err := uc.ListPlayerInteractions(ctx, 1)
		if err != nil {
			t.Fatalf("ListPlayerInteractions: %v", err)
		}
		if len(interactions) != 2 {
			t.Fatalf("len = %d, want 2 (player 1's only)", len(interactions))
		}
		// Newest first: game 3 was saved after game 1.
		if interactions[0].GameID != 3 {
			t.Errorf("interactions[0].GameID = %d, want 3 (newest first)", interactions[0].GameID)
		}
		if interactions[1].GameID != 1 {
			t.Errorf("interactions[1].GameID = %d, want 1", interactions[1].GameID)
		}
		if interactions[0].PlayerID.Int64 != 1 || interactions[1].PlayerID.Int64 != 1 {
			t.Errorf("expected only player 1's interactions, got %+v", interactions)
		}
	})
}

func TestDataFetching_ListInteractions(t *testing.T) {
	ctx := context.Background()

	t.Run("nil queries", func(t *testing.T) {
		uc := NewDataFetching(nil, newFixedClock())
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
			uc := NewDataFetching(newTestDB(t), newFixedClock())
			_, err := uc.ListInteractions(ctx, tt.gameID, tt.limit)
			requireErrContains(t, err, tt.wantErr)
		})
	}

	t.Run("game with no interactions returns empty, no error", func(t *testing.T) {
		uc := NewDataFetching(newTestDB(t), newFixedClock())
		interactions, err := uc.ListInteractions(ctx, 1, 10)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(interactions) != 0 {
			t.Errorf("expected no interactions, got %v", interactions)
		}
	})

	t.Run("limit zero is a valid boundary", func(t *testing.T) {
		uc := NewDataFetching(newTestDB(t), newFixedClock())
		if _, err := uc.ListInteractions(ctx, 1, 0); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}
