package usecase

import (
	"context"
	"fmt"
	"keep-it-up/internal/infrastructure/database"
	"strings"
)

type GameManagement struct {
	q *database.Queries
}

func NewGameManagement(q *database.Queries) *GameManagement {
	return &GameManagement{q: q}
}

func (uc *GameManagement) AddGame(ctx context.Context, name string) (database.Game, error) {
	if uc.q == nil {
		return database.Game{}, fmt.Errorf("database queries are not initialized")
	}

	name = strings.TrimSpace(name)

	if len(name) < 3 {
		return database.Game{}, fmt.Errorf(
			"Game name cannot have less than 3 characters: '%s'",
			name,
		)
	}

	game, err := uc.q.CreateGame(ctx, name)
	if err != nil {
		return database.Game{}, fmt.Errorf("failed to create game %q: %w", name, err)
	}
	return game, nil
}

func (uc *GameManagement) UpdateGame(ctx context.Context, id int64, name string) error {
	if uc.q == nil {
		return fmt.Errorf("database queries are not initialized")
	}

	if id < 1 {
		return fmt.Errorf("Invalid game ID: %d", id)
	}

	name = strings.TrimSpace(name)

	if len(name) < 3 {
		return fmt.Errorf(
			"New game name cannot have less than 3 characters: '%s'",
			name,
		)
	}

	if err := uc.q.UpdateGame(ctx, database.UpdateGameParams{
		ID:   id,
		Name: name,
	}); err != nil {
		return fmt.Errorf("failed to update game %d: %w", id, err)
	}
	return nil
}

func (uc *GameManagement) DeleteGame(ctx context.Context, id int64) error {
	if uc.q == nil {
		return fmt.Errorf("database queries are not initialized")
	}

	if id < 1 {
		return fmt.Errorf("Invalid game ID: %d", id)
	}

	if err := uc.q.DeleteGame(ctx, id); err != nil {
		return fmt.Errorf("failed to delete game %d: %w", id, err)
	}
	return nil
}
