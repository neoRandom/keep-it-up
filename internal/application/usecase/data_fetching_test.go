package usecase

import (
	"context"
	"testing"

	"keep-it-up/internal/infrastructure/constant"
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
		{"invalid player id: zero", 0, "invalid player ID"},
		{"invalid player id: negative", -1, "invalid player ID"},
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
		{"invalid game id: zero", 0, "invalid game ID"},
		{"invalid game id: negative", -1, "invalid game ID"},
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
		{"invalid game id: zero", 0, "invalid game ID"},
		{"invalid game id: negative", -1, "invalid game ID"},
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
		if interaction.Action != "saved" || interaction.OccurredAt.Format(constant.DBDatetimeFormat) != "2026-08-22T12:00:00Z" {
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
		{"invalid game id: zero", 0, "invalid game ID"},
		{"invalid game id: negative", -1, "invalid game ID"},
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
		if interaction.Action != "saved" || interaction.OccurredAt.Format(constant.DBDatetimeFormat) != "2026-08-22T12:00:03Z" {
			t.Errorf("last interaction = %+v, want the final save", interaction)
		}
	})
}

func TestDataFetching_ListPlayerInteractions(t *testing.T) {
	ctx := context.Background()

	t.Run("nil queries", func(t *testing.T) {
		uc := NewDataFetching(nil, newFixedClock())
		_, err := uc.ListPlayerInteractions(ctx, 1, 1, 10, 0)
		requireErrContains(t, err, "not initialized")
	})

	for _, tt := range []struct {
		name     string
		gameID   int64
		playerID int64
		limit    int64
		offset   int64
		wantErr  string
	}{
		{"invalid game id: zero", 0, 1, 10, 0, "invalid game ID"},
		{"invalid game id: negative", -1, 1, 10, 0, "invalid game ID"},
		{"invalid player id: zero", 1, 0, 10, 0, "invalid player ID"},
		{"invalid player id: negative", 1, -1, 10, 0, "invalid player ID"},
		{"negative limit", 1, 1, -1, 0, "query limit cannot be less than 0"},
		{"negative offset", 1, 1, 10, -1, "query offset cannot be less than 0"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewDataFetching(newTestDB(t), newFixedClock())
			_, err := uc.ListPlayerInteractions(ctx, tt.gameID, tt.playerID, tt.limit, tt.offset)
			requireErrContains(t, err, tt.wantErr)
		})
	}

	t.Run("player with no interactions in a game returns empty, no error", func(t *testing.T) {
		uc := NewDataFetching(newTestDB(t), newFixedClock())
		interactions, err := uc.ListPlayerInteractions(ctx, 5, 1, 10, 0)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(interactions) != 0 {
			t.Errorf("expected no interactions, got %v", interactions)
		}
	})

	t.Run("returns only the given player's interactions for the given game, newest first", func(t *testing.T) {
		q := newTestDB(t)
		commands := NewGameCommands(q, newFixedClock())

		// Player 1 saves in game 1, player 2 saves in game 1, player 1 saves
		// again in game 1, player 1 saves in game 2. The fixed clock advances
		// one second per call, so timestamps are strictly increasing.
		if err := commands.SaveGame(ctx, 1, 1, 60); err != nil {
			t.Fatalf("setup save 1: %v", err)
		}
		if err := commands.SaveGame(ctx, 1, 2, 60); err != nil {
			t.Fatalf("setup save 2: %v", err)
		}
		if err := commands.SaveGame(ctx, 1, 1, 60); err != nil {
			t.Fatalf("setup save 3: %v", err)
		}
		if err := commands.SaveGame(ctx, 2, 1, 60); err != nil {
			t.Fatalf("setup save 4: %v", err)
		}

		uc := NewDataFetching(q, newFixedClock())
		interactions, err := uc.ListPlayerInteractions(ctx, 1, 1, 10, 0)
		if err != nil {
			t.Fatalf("ListPlayerInteractions: %v", err)
		}
		if len(interactions) != 2 {
			t.Fatalf("len = %d, want 2 (player 1's in game 1 only)", len(interactions))
		}
		// Newest first within game 1: the second save precedes the first.
		if interactions[0].GameID != 1 || interactions[1].GameID != 1 {
			t.Errorf("expected only game 1 interactions, got %+v", interactions)
		}
		if *interactions[0].PlayerID != 1 || *interactions[1].PlayerID != 1 {
			t.Errorf("expected only player 1's interactions, got %+v", interactions)
		}
	})

	t.Run("offset paginates past earlier interactions for the player and game", func(t *testing.T) {
		q := newTestDB(t)
		commands := NewGameCommands(q, newFixedClock())

		// Player 1 saves in game 1 three times (12:00:00, 12:00:01, 12:00:02).
		// The fixed clock advances one second per call.
		for i := 0; i < 3; i++ {
			if err := commands.SaveGame(ctx, 1, 1, 60); err != nil {
				t.Fatalf("setup save %d: %v", i+1, err)
			}
		}

		uc := NewDataFetching(q, newFixedClock())
		interactions, err := uc.ListPlayerInteractions(ctx, 1, 1, 10, 1)
		if err != nil {
			t.Fatalf("ListPlayerInteractions: %v", err)
		}
		if len(interactions) != 2 {
			t.Fatalf("len = %d, want 2 (one skipped by offset)", len(interactions))
		}
		// Newest first: with offset 1, the most recent (12:00:02) is skipped.
		if interactions[0].OccurredAt.Format(constant.DBDatetimeFormat) != "2026-08-22T12:00:01Z" {
			t.Errorf("interactions[0].OccurredAt.Format(constant.DBDatetimeFormat) = %q, want 12:00:01", interactions[0].OccurredAt.Format(constant.DBDatetimeFormat))
		}
	})
}

func TestDataFetching_ListInteractions(t *testing.T) {
	ctx := context.Background()

	t.Run("nil queries", func(t *testing.T) {
		uc := NewDataFetching(nil, newFixedClock())
		_, err := uc.ListInteractions(ctx, 1, 10, 0)
		requireErrContains(t, err, "not initialized")
	})

	for _, tt := range []struct {
		name    string
		gameID  int64
		limit   int64
		offset  int64
		wantErr string
	}{
		{"invalid game id: zero", 0, 10, 0, "invalid game ID"},
		{"invalid game id: negative", -1, 10, 0, "invalid game ID"},
		{"negative limit", 1, -1, 0, "query limit cannot be less than 0"},
		{"negative offset", 1, 10, -1, "query offset cannot be less than 0"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewDataFetching(newTestDB(t), newFixedClock())
			_, err := uc.ListInteractions(ctx, tt.gameID, tt.limit, tt.offset)
			requireErrContains(t, err, tt.wantErr)
		})
	}

	t.Run("game with no interactions returns empty, no error", func(t *testing.T) {
		uc := NewDataFetching(newTestDB(t), newFixedClock())
		interactions, err := uc.ListInteractions(ctx, 1, 10, 0)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(interactions) != 0 {
			t.Errorf("expected no interactions, got %v", interactions)
		}
	})

	t.Run("limit zero is a valid boundary", func(t *testing.T) {
		uc := NewDataFetching(newTestDB(t), newFixedClock())
		if _, err := uc.ListInteractions(ctx, 1, 0, 0); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("offset paginates past earlier interactions", func(t *testing.T) {
		q := newTestDB(t)
		commands := NewGameCommands(q, newFixedClock())

		// Three saves, one second apart: 12:00:00, 12:00:01, 12:00:02.
		for i := 0; i < 3; i++ {
			if err := commands.SaveGame(ctx, 5, 1, 60); err != nil {
				t.Fatalf("setup save %d: %v", i+1, err)
			}
		}

		uc := NewDataFetching(q, newFixedClock())
		interactions, err := uc.ListInteractions(ctx, 5, 10, 1)
		if err != nil {
			t.Fatalf("ListInteractions: %v", err)
		}
		if len(interactions) != 2 {
			t.Fatalf("len = %d, want 2 (one skipped by offset)", len(interactions))
		}
		// Newest first: with offset 1, the most recent (12:00:02) is skipped.
		if interactions[0].OccurredAt.Format(constant.DBDatetimeFormat) != "2026-08-22T12:00:01Z" {
			t.Errorf("interactions[0].OccurredAt.Format(constant.DBDatetimeFormat) = %q, want 12:00:01", interactions[0].OccurredAt.Format(constant.DBDatetimeFormat))
		}
	})
}
