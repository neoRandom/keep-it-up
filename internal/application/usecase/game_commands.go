package usecase

import (
	"context"
	"time"
)

type GameCommands struct{}

func NewGameCommands() *GameCommands {
	return &GameCommands{}
}

func (uc *GameCommands) SaveGame(
	ctx context.Context, gameId int64, playerId int64, amount time.Time,
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
