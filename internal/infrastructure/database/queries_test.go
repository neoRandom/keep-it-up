package database_test

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	database "keep-it-up/internal/infrastructure/database"
	"keep-it-up/internal/testutil"
)


// newTestQueries is a thin wrapper over testutil.NewTestDB; the shared schema
// and migration setup lives in internal/testutil.
func newTestQueries(t *testing.T) *database.Queries {
	return testutil.NewTestDB(t)
}

func wantErrContaining(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Errorf("expected error containing %q, got %q", want, err.Error())
	}
}

// seedGame creates a game row, required by the access and interaction FKs.
func seedGame(t *testing.T, ctx context.Context, q *database.Queries, name string) database.Game {
	t.Helper()
	game, err := q.CreateGame(ctx, name)
	if err != nil {
		t.Fatalf("CreateGame(%q): %v", name, err)
	}
	return game
}

// seedPlayer creates a player row, required by the access FK.
func seedPlayer(t *testing.T, ctx context.Context, q *database.Queries, username string) database.Player {
	t.Helper()
	player, err := q.CreatePlayer(ctx, database.CreatePlayerParams{
		Name:           username,
		Username:       username,
		HashedPassword: "hash",
	})
	if err != nil {
		t.Fatalf("CreatePlayer(%q): %v", username, err)
	}
	return player
}


func TestGameCRUD(t *testing.T) {
	ctx := context.Background()
	q := newTestQueries(t)

	game := seedGame(t, ctx, q, "Alpha")
	if name, ok := testutil.ReadGameNameByAccess(t, ctx, q, game.ID); !ok || name != "Alpha" {
		t.Fatalf("read after create: name=%q ok=%v, want Alpha", name, ok)
	}

	if err := q.UpdateGame(ctx, database.UpdateGameParams{ID: game.ID, Name: "Beta"}); err != nil {
		t.Fatalf("UpdateGame: %v", err)
	}
	if name, ok := testutil.ReadGameNameByAccess(t, ctx, q, game.ID); !ok || name != "Beta" {
		t.Fatalf("read after update: name=%q ok=%v, want Beta", name, ok)
	}

	if err := q.DeleteGame(ctx, game.ID); err != nil {
		t.Fatalf("DeleteGame: %v", err)
	}
	if _, ok := testutil.ReadGameNameByAccess(t, ctx, q, game.ID); ok {
		t.Fatal("ListPlayerGames found deleted game")
	}
}

func TestPlayerAccessLifecycle(t *testing.T) {
	ctx := context.Background()
	q := newTestQueries(t)

	game := seedGame(t, ctx, q, "Alpha")
	player := seedPlayer(t, ctx, q, "neo")

	grant := func() error {
		_, err := q.GrantPlayerAccess(ctx, database.GrantPlayerAccessParams{GameID: game.ID, PlayerID: player.ID})
		return err
	}
	hasAccess := func() bool {
		ok, err := q.CheckPlayerAccess(ctx, database.CheckPlayerAccessParams{GameID: game.ID, PlayerID: player.ID})
		if err != nil {
			t.Fatalf("CheckPlayerAccess: %v", err)
		}
		return ok
	}

	if hasAccess() {
		t.Fatal("access granted before GrantPlayerAccess")
	}
	if err := grant(); err != nil {
		t.Fatalf("GrantPlayerAccess: %v", err)
	}
	if !hasAccess() {
		t.Fatal("access not granted after GrantPlayerAccess")
	}

	games, err := q.ListPlayerGames(ctx, player.ID)
	if err != nil {
		t.Fatalf("ListPlayerGames: %v", err)
	}
	if len(games) != 1 || games[0].ID != game.ID {
		t.Fatalf("ListPlayerGames = %+v, want the granted game", games)
	}

	if err := q.RevokePlayerAccess(ctx, database.RevokePlayerAccessParams{GameID: game.ID, PlayerID: player.ID}); err != nil {
		t.Fatalf("RevokePlayerAccess: %v", err)
	}
	if hasAccess() {
		t.Fatal("access still granted after RevokePlayerAccess")
	}
}

// isoTime formats a base time into the RFC3339 format the migration's
// valid_occurred_at_iso constraint requires, advancing by step each call so
// timestamps stay strictly increasing for the state-machine trigger.
func isoTime(base time.Time, step int) string {
	return base.Add(time.Duration(step) * time.Second).Format("2006-01-02T15:04:05Z07:00")
}

func TestStateMachine_ValidSequencePersists(t *testing.T) {
	ctx := context.Background()
	q := newTestQueries(t)
	base := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)

	insertSave(t, ctx, q, 1, 1, isoTime(base, 1), 60)
	pause, err := q.PauseGame(ctx, database.PauseGameParams{GameID: 1, PlayerID: sql.NullInt64{Int64: 1, Valid: true}, OccurredAt: isoTime(base, 2)})
	if err != nil {
		t.Fatalf("PauseGame after save: %v", err)
	}
	resume, err := q.ResumeGame(ctx, database.ResumeGameParams{GameID: 1, PlayerID: sql.NullInt64{Int64: 1, Valid: true}, OccurredAt: isoTime(base, 3)})
	if err != nil {
		t.Fatalf("ResumeGame after pause: %v", err)
	}
	if pause.Action != "paused" || resume.Action != "resumed" {
		t.Fatalf("expected paused/resumed actions, got %q/%q", pause.Action, resume.Action)
	}

	rows, err := q.ListInteractionsForReplay(ctx, 1)
	if err != nil {
		t.Fatalf("ListInteractionsForReplay: %v", err)
	}
	actions := make([]string, len(rows))
	for i, r := range rows {
		actions[i] = r.Action
	}
	want := []string{"saved", "paused", "resumed"}
	if strings.Join(actions, ",") != strings.Join(want, ",") {
		t.Fatalf("actions = %v, want %v", actions, want)
	}
}

func TestStateMachine_RejectsSaveWhilePaused(t *testing.T) {
	ctx := context.Background()
	q := newTestQueries(t)
	base := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)

	insertSave(t, ctx, q, 1, 1, isoTime(base, 1), 60)
	if _, err := q.PauseGame(ctx, database.PauseGameParams{GameID: 1, PlayerID: sql.NullInt64{Int64: 1, Valid: true}, OccurredAt: isoTime(base, 2)}); err != nil {
		t.Fatalf("setup PauseGame: %v", err)
	}
	_, err := q.SaveGame(ctx, database.SaveGameParams{GameID: 1, PlayerID: sql.NullInt64{Int64: 1, Valid: true}, OccurredAt: isoTime(base, 3), SavedBy: sql.NullInt64{Int64: 30, Valid: true}})
	wantErrContaining(t, err, "cannot save while paused")
}

func TestStateMachine_RejectsPauseWhenNotPlaying(t *testing.T) {
	ctx := context.Background()
	q := newTestQueries(t)
	base := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)

	// A game with no prior interaction is "not_started"; pausing must be
	// rejected by the state machine (not by schema validation).
	_, err := q.PauseGame(ctx, database.PauseGameParams{GameID: 1, PlayerID: sql.NullInt64{Int64: 1, Valid: true}, OccurredAt: isoTime(base, 1)})
	wantErrContaining(t, err, "cannot pause")
}

func TestStateMachine_RejectsResumeWhenNotPaused(t *testing.T) {
	ctx := context.Background()
	q := newTestQueries(t)
	base := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)

	insertSave(t, ctx, q, 1, 1, isoTime(base, 1), 60)
	_, err := q.ResumeGame(ctx, database.ResumeGameParams{GameID: 1, PlayerID: sql.NullInt64{Int64: 1, Valid: true}, OccurredAt: isoTime(base, 2)})
	wantErrContaining(t, err, "cannot resume")
}

func insertSave(t *testing.T, ctx context.Context, q *database.Queries, gameID, playerID int64, occurredAt string, savedBy int64) database.Interaction {
	t.Helper()
	row, err := q.SaveGame(ctx, database.SaveGameParams{
		GameID:     gameID,
		PlayerID:   sql.NullInt64{Int64: playerID, Valid: true},
		OccurredAt: occurredAt,
		SavedBy:    sql.NullInt64{Int64: savedBy, Valid: true},
	})
	if err != nil {
		t.Fatalf("SaveGame: %v", err)
	}
	return row
}
