package usecase

import (
	"context"
	"fmt"
	"keep-it-up/internal/infrastructure/database"
	"keep-it-up/internal/infrastructure/util"
)

type GameCommands struct {
	q *database.Queries
}

func NewGameCommands() *GameCommands {
	return &GameCommands{}
}

func (uc *GameCommands) AddGame(ctx context.Context, name string) (database.Game, error) {
	if len(name) < 3 {
		return database.Game{}, fmt.Errorf("Game name cannot have less than 3 characters: '%s'", name)
	}

	if !util.IsAlphanumeric(name) {
		return database.Game{}, fmt.Errorf("Game name isn't purely alphanumeric: '%s'", name)
	}

	return uc.q.CreateGame(ctx, name)
}

func (uc *GameCommands) UpdateGame(ctx context.Context, id int64, name string) error {
	if id < 1 {
		return fmt.Errorf("Invalid game ID: %d", id)
	}
	
	if len(name) < 3 {
		return fmt.Errorf("New game name cannot have less than 3 characters: '%s'", name)
	}

	if !util.IsAlphanumeric(name) {
		return fmt.Errorf("New game name isn't purely alphanumeric: '%s'", name)
	}

	return uc.q.UpdateGame(ctx, database.UpdateGameParams{
		ID: id,
		Name: name,
	})
}

func (uc *GameCommands) DeleteGame(ctx context.Context, id int64) error {
	return nil
}
