package usecase

import (
	"context"
	"errors"
	"fmt"
	"keep-it-up/internal/core/model"
	"keep-it-up/internal/core/service"
	"keep-it-up/internal/infrastructure/database"
)

type DataFetching struct {
	q *database.Queries
}

func NewDataFetching(q *database.Queries) *DataFetching {
	return &DataFetching{q: q}
}

func (uc *DataFetching) ListPlayerGames(
	ctx context.Context, playerId int64,
) ([]database.Game, error) {
	if uc.q == nil {
		return nil, errors.New("database queries are not initialized")
	}

	if playerId < 1 {
		return nil, fmt.Errorf("Invalid player ID: %d", playerId)
	}

	return uc.q.ListPlayerGames(ctx, playerId)
}

func (uc *DataFetching) GetSharedData(
	ctx context.Context, gameId int64,
) (*model.SharedData, error) {
	if uc.q == nil {
		return nil, errors.New("database queries are not initialized")
	}

	if gameId < 1 {
		return nil, fmt.Errorf("Invalid game ID: %d", gameId)
	}

	interactions, err := uc.q.ListInteractionsForReplay(ctx, gameId)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to list interactions for replay: %w", err,
		)
	}

	return service.BuildSharedData(gameId, interactions)
}

func (uc *DataFetching) ListInteractions(
	ctx context.Context, gameId int64, limit int64,
) ([]database.Interaction, error) {
	if uc.q == nil {
		return nil, errors.New("database queries are not initialized")
	}

	if gameId < 1 {
		return nil, fmt.Errorf("Invalid game ID: %d", gameId)
	}

	if limit < 0 {
		return nil, errors.New("query limit cannot be less than 0")
	}

	return uc.q.ListRecentInteractions(
		ctx,
		database.ListRecentInteractionsParams{
			GameID: gameId,
			Limit:  limit,
		},
	)
}
