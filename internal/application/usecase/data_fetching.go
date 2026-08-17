package usecase

import (
	"context"
	"keep-it-up/internal/core/model"
	"keep-it-up/internal/infrastructure/database"
)

type DataFetching struct{}

func NewDataFetching() *DataFetching {
	return &DataFetching{}
}

func (uc *DataFetching) ListPlayerGames(
	ctx context.Context, playerId int64,
) ([]database.Game, error) {
	return nil, nil
}

func (uc *DataFetching) GetSharedData(
	ctx context.Context, gameId int64,
) (model.SharedData, error) {
	return model.SharedData{}, nil
}

func (uc *DataFetching) ListInteractions(
	ctx context.Context, gameId int64, count int,
) ([]database.Interaction, error) {
	return nil, nil
}
