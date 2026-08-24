package usecase

import (
	"context"
	"strings"
	"testing"
	"time"
)

// fixedClock is a TimeProvider that returns a fixed point in time plus a
// configurable step on each call. It lets tests drive the state machine with
// deterministic, monotonically increasing `occurred_at` timestamps (the SQLite
// trigger rejects out-of-order inserts for the same game).
type fixedClock struct {
	now  time.Time
	step time.Duration
}

func (c *fixedClock) Time() (time.Time, error) {
	t := c.now
	c.now = c.now.Add(c.step)
	return t, nil
}

func newFixedClock() *fixedClock {
	return &fixedClock{
		now:  time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC),
		step: time.Second,
	}
}

func TestGameCommands_ValidSavePauseResumeSequence(t *testing.T) {
	ctx := context.Background()
	uc := mustNewGameCommands(t, newTestDB(t), newFixedClock())

	// A fresh game starts in "not_started"; a save starts it.
	if err := uc.SaveGame(ctx, 1, 1, 60); err != nil {
		t.Fatalf("SaveGame(initial): %v", err)
	}
	// Pause requires the game to be playing (last action saved).
	if err := uc.PauseGame(ctx, 1, 1); err != nil {
		t.Fatalf("PauseGame: %v", err)
	}
	// Resume requires the game to be paused.
	if err := uc.ResumeGame(ctx, 1, 1); err != nil {
		t.Fatalf("ResumeGame: %v", err)
	}

	// The sequence must have persisted exactly three interactions, in order.
	// ListInteractions orders by occurred_at DESC, id DESC, so the last item is
	// the oldest (the initial save). A limit of 0 would produce zero rows, so
	// request a positive limit.
	fetch := mustNewDataFetching(t, uc.q, newFixedClock())
	interactions, err := fetch.ListInteractions(ctx, 1, 10, 0)
	if err != nil {
		t.Fatalf("ListInteractions: %v", err)
	}
	if len(interactions) != 3 {
		t.Fatalf("expected 3 interactions after save/pause/resume, got %d", len(interactions))
	}
	want := []string{"saved", "paused", "resumed"}
	got := []string{
		interactions[2].Action,
		interactions[1].Action,
		interactions[0].Action,
	}
	for i, a := range want {
		if got[i] != a {
			t.Errorf("interaction %d action = %q, want %q", i, got[i], a)
		}
	}
}

func TestGameCommands_SaveWhilePausedRejected(t *testing.T) {
	ctx := context.Background()
	uc := mustNewGameCommands(t, newTestDB(t), newFixedClock())

	if err := uc.SaveGame(ctx, 1, 1, 60); err != nil {
		t.Fatalf("SaveGame(initial): %v", err)
	}
	if err := uc.PauseGame(ctx, 1, 1); err != nil {
		t.Fatalf("PauseGame: %v", err)
	}

	err := uc.SaveGame(ctx, 1, 1, 60)
	if err == nil {
		t.Fatal("SaveGame while paused should be rejected")
	}
	if !strings.Contains(err.Error(), "cannot save while paused") {
		t.Errorf("expected 'cannot save while paused', got %q", err.Error())
	}
}

func TestGameCommands_PauseWhenNotPlayingRejected(t *testing.T) {
	ctx := context.Background()
	uc := mustNewGameCommands(t, newTestDB(t), newFixedClock())

	// A game with no interactions is "not_started"; pausing must be rejected.
	err := uc.PauseGame(ctx, 1, 1)
	if err == nil {
		t.Fatal("PauseGame on a not-started game should be rejected")
	}
	if !strings.Contains(err.Error(), "cannot pause") {
		t.Errorf("expected 'cannot pause', got %q", err.Error())
	}
}

func TestGameCommands_ResumeWhenNotPausedRejected(t *testing.T) {
	ctx := context.Background()
	uc := mustNewGameCommands(t, newTestDB(t), newFixedClock())

	if err := uc.SaveGame(ctx, 1, 1, 60); err != nil {
		t.Fatalf("SaveGame(initial): %v", err)
	}

	err := uc.ResumeGame(ctx, 1, 1)
	if err == nil {
		t.Fatal("ResumeGame from a playing state should be rejected")
	}
	if !strings.Contains(err.Error(), "cannot resume") {
		t.Errorf("expected 'cannot resume', got %q", err.Error())
	}
}

func TestGameCommands_InvalidInputs(t *testing.T) {
	ctx := context.Background()
	uc := mustNewGameCommands(t, newTestDB(t), newFixedClock())

	for _, gameID := range []int64{0, -1} {
		if err := uc.SaveGame(ctx, gameID, 1, 60); err == nil {
			t.Errorf("SaveGame accepted invalid gameId=%d", gameID)
		}
		if err := uc.PauseGame(ctx, gameID, 1); err == nil {
			t.Errorf("PauseGame accepted invalid gameId=%d", gameID)
		}
		if err := uc.ResumeGame(ctx, gameID, 1); err == nil {
			t.Errorf("ResumeGame accepted invalid gameId=%d", gameID)
		}
	}

	for _, playerID := range []int64{0, -1} {
		if err := uc.SaveGame(ctx, 1, playerID, 60); err == nil {
			t.Errorf("SaveGame accepted invalid playerId=%d", playerID)
		}
		if err := uc.PauseGame(ctx, 1, playerID); err == nil {
			t.Errorf("PauseGame accepted invalid playerId=%d", playerID)
		}
		if err := uc.ResumeGame(ctx, 1, playerID); err == nil {
			t.Errorf("ResumeGame accepted invalid playerId=%d", playerID)
		}
	}

	for _, duration := range []int64{0, -5} {
		if err := uc.SaveGame(ctx, 1, 1, duration); err == nil {
			t.Errorf("SaveGame accepted invalid duration=%d", duration)
		}
	}
}

func TestGameCommands_NilDependencies(t *testing.T) {
	if _, err := NewGameCommands(nil, newFixedClock()); err == nil {
		t.Fatal("NewGameCommands() should return an error for nil queries")
	}
	if _, err := NewGameCommands(newTestDB(t), nil); err == nil {
		t.Fatal("NewGameCommands() should return an error for nil time provider")
	}
}
