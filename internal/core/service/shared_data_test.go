package service

import (
	"testing"
	"time"

	"keep-it-up/internal/core/model"
)

func TestBuildSharedData_ComputesValidFromDeadline(t *testing.T) {
	// A single save at 12:00:00 UTC with a 60s duration sets the deadline to
	// 12:01:00 UTC.
	interactions := []model.Interaction{
		{
			ID:         1,
			GameID:     5,
			PlayerID:   intPtr(1),
			Action:     "saved",
			OccurredAt: time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC),
			SavedBy:    intPtr(60),
		},
	}

	t.Run("now before deadline -> valid true", func(t *testing.T) {
		shared, err := BuildSharedData(5, interactions, time.Date(2026, 8, 22, 12, 0, 30, 0, time.UTC))
		if err != nil {
			t.Fatalf("BuildSharedData: %v", err)
		}
		if shared.Valid == nil || !*shared.Valid {
			t.Errorf("Valid = %v, want true", shared.Valid)
		}
	})

	t.Run("now after deadline -> valid false", func(t *testing.T) {
		shared, err := BuildSharedData(5, interactions, time.Date(2026, 8, 22, 12, 5, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("BuildSharedData: %v", err)
		}
		if shared.Valid == nil || *shared.Valid {
			t.Errorf("Valid = %v, want false", shared.Valid)
		}
	})

	t.Run("not started -> no deadline, valid nil", func(t *testing.T) {
		shared, err := BuildSharedData(5, nil, time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("BuildSharedData: %v", err)
		}
		if shared.Status != model.NotStarted || shared.Valid != nil || shared.DeadlineAt != nil {
			t.Errorf("expected not_started with nil valid/deadline, got %+v", shared)
		}
	})
}

func TestBuildSharedData_PausedGameStaysValid(t *testing.T) {
	// Save at 12:00:00 with 60s => deadline 12:01:00; pause at 12:00:30.
	// LastPausedAt (12:00:30) is after LastSavedAt (12:00:00) and before
	// DeadlineAt (12:01:00) with Status == paused, so the game is valid
	// even when "now" is far past the deadline.
	interactions := []model.Interaction{
		{
			ID:         1,
			GameID:     5,
			PlayerID:   intPtr(1),
			Action:     "saved",
			OccurredAt: time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC),
			SavedBy:    intPtr(60),
		},
		{
			ID:         2,
			GameID:     5,
			PlayerID:   intPtr(1),
			Action:     "paused",
			OccurredAt: time.Date(2026, 8, 22, 12, 0, 30, 0, time.UTC),
		},
	}

	t.Run("paused and now past deadline -> valid true", func(t *testing.T) {
		shared, err := BuildSharedData(5, interactions, time.Date(2026, 8, 22, 12, 5, 0, 0, time.UTC))
		if err != nil {
			t.Fatalf("BuildSharedData: %v", err)
		}
		if shared.Status != model.Paused {
			t.Errorf("Status = %q, want paused", shared.Status)
		}
		if shared.Valid == nil || !*shared.Valid {
			t.Errorf("Valid = %v, want true while paused", shared.Valid)
		}
	})

	t.Run("paused and now before deadline -> valid true", func(t *testing.T) {
		shared, err := BuildSharedData(5, interactions, time.Date(2026, 8, 22, 12, 0, 45, 0, time.UTC))
		if err != nil {
			t.Fatalf("BuildSharedData: %v", err)
		}
		if shared.Valid == nil || !*shared.Valid {
			t.Errorf("Valid = %v, want true", shared.Valid)
		}
	})
}

func TestBuildSharedData_PlayingAfterDeadlineInvalid(t *testing.T) {
	// A game that is playing (not paused) with "now" past the deadline must
	// remain invalid — the paused exception must not leak into other states.
	interactions := []model.Interaction{
		{
			ID:         1,
			GameID:     5,
			PlayerID:   intPtr(1),
			Action:     "saved",
			OccurredAt: time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC),
			SavedBy:    intPtr(60),
		},
	}

	shared, err := BuildSharedData(5, interactions, time.Date(2026, 8, 22, 12, 5, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("BuildSharedData: %v", err)
	}
	if shared.Status != model.Playing {
		t.Errorf("Status = %q, want playing", shared.Status)
	}
	if shared.Valid == nil || *shared.Valid {
		t.Errorf("Valid = %v, want false for playing game past deadline", shared.Valid)
	}
}

func TestComputeValid_PausedBoundaryConditions(t *testing.T) {
	savedAt := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	deadline := time.Date(2026, 8, 22, 12, 1, 0, 0, time.UTC)
	now := time.Date(2026, 8, 22, 12, 5, 0, 0, time.UTC) // past deadline

	pausedAt := savedAt.Add(30 * time.Second) // 12:00:30
	base := &model.SharedData{
		Status:       model.Paused,
		DeadlineAt:   &deadline,
		LastSavedAt:  &savedAt,
		LastPausedAt: &pausedAt,
	}

	t.Run("condition met -> valid true", func(t *testing.T) {
		s := *base
		ComputeValid(&s, now)
		if s.Valid == nil || !*s.Valid {
			t.Errorf("Valid = %v, want true when paused between save and deadline", s.Valid)
		}
	})

	t.Run("status not paused -> falls back to deadline check", func(t *testing.T) {
		s := *base
		s.Status = model.Playing
		ComputeValid(&s, now)
		if s.Valid == nil || *s.Valid {
			t.Errorf("Valid = %v, want false when not paused and now past deadline", s.Valid)
		}
	})

	t.Run("LastPausedAt not before DeadlineAt -> falls back to deadline check", func(t *testing.T) {
		s := *base
		atDeadline := *s.DeadlineAt
		s.LastPausedAt = &atDeadline // == deadline, not < deadline
		ComputeValid(&s, now)
		if s.Valid == nil || *s.Valid {
			t.Errorf("Valid = %v, want false when pause not before deadline", s.Valid)
		}
	})

	t.Run("LastPausedAt not after LastSavedAt -> falls back to deadline check", func(t *testing.T) {
		s := *base
		s.LastPausedAt = &savedAt // == LastSavedAt, not > LastSavedAt
		ComputeValid(&s, now)
		if s.Valid == nil || *s.Valid {
			t.Errorf("Valid = %v, want false when pause not after last save", s.Valid)
		}
	})
}

func intPtr(v int64) *int64 {
	return &v
}
