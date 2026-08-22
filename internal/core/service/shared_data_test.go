package service

import (
	"database/sql"
	"testing"
	"time"

	"keep-it-up/internal/core/model"
	"keep-it-up/internal/infrastructure/database"
)

func TestBuildSharedData_ComputesValidFromDeadline(t *testing.T) {
	// A single save at 12:00:00 UTC with a 60s duration sets the deadline to
	// 12:01:00 UTC.
	interactions := []database.Interaction{
		{
			ID:         1,
			GameID:     5,
			PlayerID:   sql.NullInt64{Int64: 1, Valid: true},
			Action:     "saved",
			OccurredAt: "2026-08-22T12:00:00Z",
			SavedBy:    sql.NullInt64{Int64: 60, Valid: true},
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