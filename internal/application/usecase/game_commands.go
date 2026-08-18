package usecase

import (
	"context"
)

type GameCommands struct{}

func NewGameCommands() *GameCommands {
	return &GameCommands{}
}

func (uc *GameCommands) SaveGame(
	ctx context.Context, gameId int64, playerId int64, duration int64,
) error {
	return nil
}

func (uc *GameCommands) ResumeGame(
	ctx context.Context, gameId int64, playerId int64,
) error {
	return nil
}

func (uc *GameCommands) PauseGame(
	ctx context.Context, gameId int64, playerId int64,
) error {
	return nil
}
