package usecase

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"keep-it-up/internal/core/model"
	"keep-it-up/internal/core/port"
	"keep-it-up/internal/core/service"
	"keep-it-up/internal/infrastructure/database"
)

type DataFetching struct {
	q  *database.Queries
	tp port.TimeProvider
}

func NewDataFetching(q *database.Queries, tp port.TimeProvider) *DataFetching {
	return &DataFetching{q: q, tp: tp}
}

func (uc *DataFetching) ListPlayerGames(
	ctx context.Context, playerId int64,
) ([]database.Game, error) {
	if uc.q == nil {
		return nil, errors.New("database queries are not initialized")
	}

	if playerId < 1 {
		return nil, fmt.Errorf("invalid player ID: %d", playerId)
	}

	games, err := uc.q.ListPlayerGames(ctx, playerId)
	if err != nil {
		return nil, fmt.Errorf("failed to list games for player %d: %w", playerId, err)
	}
	return games, nil
}

func (uc *DataFetching) GetSharedData(
	ctx context.Context, gameId int64,
) (*model.SharedData, error) {
	if uc.q == nil {
		return nil, errors.New("database queries are not initialized")
	}

	if uc.tp == nil {
		return nil, errors.New("time provider is not initialized")
	}

	if gameId < 1 {
		return nil, fmt.Errorf("invalid game ID: %d", gameId)
	}

	interactions, err := uc.q.ListInteractionsForReplay(ctx, gameId)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to list interactions for replay: %w", err,
		)
	}

	now, err := uc.tp.Time()
	if err != nil {
		return nil, fmt.Errorf("failed to get current time: %w", err)
	}

	return service.BuildSharedData(gameId, interactions, now)
}

func (uc *DataFetching) ListInteractions(
	ctx context.Context, gameId, limit, offset int64,
) ([]database.Interaction, error) {
	if uc.q == nil {
		return nil, errors.New("database queries are not initialized")
	}

	if gameId < 1 {
		return nil, fmt.Errorf("invalid game ID: %d", gameId)
	}

	if limit < 0 {
		return nil, errors.New("query limit cannot be less than 0")
	}

	if offset < 0 {
		return nil, errors.New("query offset cannot be less than 0")
	}

	interactions, err := uc.q.ListRecentInteractions(
		ctx,
		database.ListRecentInteractionsParams{
			GameID: gameId,
			Limit:  limit,
			Offset: offset,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list interactions for game %d: %w", gameId, err)
	}
	return interactions, nil
}

// ListPlayerInteractions returns the interactions a player made in a specific
// game, newest first, limited and paginated by offset.
func (uc *DataFetching) ListPlayerInteractions(
	ctx context.Context, gameId, playerId, limit, offset int64,
) ([]database.Interaction, error) {
	if uc.q == nil {
		return nil, errors.New("database queries are not initialized")
	}

	if gameId < 1 {
		return nil, fmt.Errorf("invalid game ID: %d", gameId)
	}

	if playerId < 1 {
		return nil, fmt.Errorf("invalid player ID: %d", playerId)
	}

	if limit < 0 {
		return nil, errors.New("query limit cannot be less than 0")
	}

	if offset < 0 {
		return nil, errors.New("query offset cannot be less than 0")
	}

	interactions, err := uc.q.ListPlayerInteractions(
		ctx,
		database.ListPlayerInteractionsParams{
			GameID:   gameId,
			PlayerID: sql.NullInt64{Int64: playerId, Valid: true},
			Limit:    limit,
			Offset:   offset,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list interactions for player %d in game %d: %w", playerId, gameId, err)
	}
	return interactions, nil
}

// FirstInteraction returns the earliest interaction of a game, or nil if the
// game has no interactions yet (i.e. it has never been started).
func (uc *DataFetching) FirstInteraction(
	ctx context.Context, gameId int64,
) (*database.Interaction, error) {
	if uc.q == nil {
		return nil, errors.New("database queries are not initialized")
	}

	if gameId < 1 {
		return nil, fmt.Errorf("invalid game ID: %d", gameId)
	}

	interaction, err := uc.q.FirstInteraction(ctx, gameId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get first interaction for game %d: %w", gameId, err)
	}
	return &interaction, nil
}

// LastInteraction returns the latest interaction of a game, or nil if the game
// has no interactions yet (i.e. it has never been started).
func (uc *DataFetching) LastInteraction(
	ctx context.Context, gameId int64,
) (*database.Interaction, error) {
	if uc.q == nil {
		return nil, errors.New("database queries are not initialized")
	}

	if gameId < 1 {
		return nil, fmt.Errorf("invalid game ID: %d", gameId)
	}

	interaction, err := uc.q.LastInteraction(ctx, gameId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get last interaction for game %d: %w", gameId, err)
	}
	return &interaction, nil
}
