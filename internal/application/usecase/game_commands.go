package usecase

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"keep-it-up/internal/core/port"
	"keep-it-up/internal/infrastructure/constant"
	"keep-it-up/internal/infrastructure/database"
)

type GameCommands struct {
	q  *database.Queries
	tp port.TimeProvider
}

func NewGameCommands(q *database.Queries, tp port.TimeProvider) *GameCommands {
	return &GameCommands{
		q: q, tp: tp,
	}
}

func (uc *GameCommands) SaveGame(
	ctx context.Context, gameId int64, playerId int64, duration int64,
) error {
	if uc.q == nil {
		return errors.New("database queries are not initialized")
	}

	if uc.tp == nil {
		return errors.New("time provider is not initialized")
	}

	if gameId < 1 {
		return fmt.Errorf("invalid game ID: %d", gameId)
	}

	if playerId < 1 {
		return fmt.Errorf("invalid player ID: %d", playerId)
	}

	if duration < 1 {
		return errors.New("save duration cannot be less than 1 second")
	}

	t, err := uc.tp.Time()
	if err != nil {
		return fmt.Errorf("failed to get current time: %w", err)
	}

	_, err = uc.q.SaveGame(ctx, database.SaveGameParams{
		GameID:     gameId,
		PlayerID:   sql.NullInt64{Int64: playerId, Valid: true},
		OccurredAt: t.Format(constant.DBDatetimeFormat),
		SavedBy:    sql.NullInt64{Int64: duration, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to save game %d: %w", gameId, err)
	}

	return nil
}

func (uc *GameCommands) ResumeGame(
	ctx context.Context, gameId int64, playerId int64,
) error {
	if uc.q == nil {
		return errors.New("database queries are not initialized")
	}

	if uc.tp == nil {
		return errors.New("time provider is not initialized")
	}

	if gameId < 1 {
		return fmt.Errorf("invalid game ID: %d", gameId)
	}

	if playerId < 1 {
		return fmt.Errorf("invalid player ID: %d", playerId)
	}

	t, err := uc.tp.Time()
	if err != nil {
		return fmt.Errorf("failed to get current time: %w", err)
	}

	_, err = uc.q.ResumeGame(ctx, database.ResumeGameParams{
		GameID:     gameId,
		PlayerID:   sql.NullInt64{Int64: playerId, Valid: true},
		OccurredAt: t.Format(constant.DBDatetimeFormat),
	})
	if err != nil {
		return fmt.Errorf("failed to resume game %d: %w", gameId, err)
	}

	return nil
}

func (uc *GameCommands) PauseGame(
	ctx context.Context, gameId int64, playerId int64,
) error {
	if uc.q == nil {
		return errors.New("database queries are not initialized")
	}

	if uc.tp == nil {
		return errors.New("time provider is not initialized")
	}

	if gameId < 1 {
		return fmt.Errorf("invalid game ID: %d", gameId)
	}

	if playerId < 1 {
		return fmt.Errorf("invalid player ID: %d", playerId)
	}

	t, err := uc.tp.Time()
	if err != nil {
		return fmt.Errorf("failed to get current time: %w", err)
	}

	_, err = uc.q.PauseGame(ctx, database.PauseGameParams{
		GameID:     gameId,
		PlayerID:   sql.NullInt64{Int64: playerId, Valid: true},
		OccurredAt: t.Format(constant.DBDatetimeFormat),
	})
	if err != nil {
		return fmt.Errorf("failed to pause game %d: %w", gameId, err)
	}

	return nil
}
