package usecase

import (
	"context"
	"errors"
	"fmt"
	"keep-it-up/internal/core/model"
	"keep-it-up/internal/infrastructure/database"
	"keep-it-up/internal/infrastructure/database/mapping"
	"strings"
)

type GameManagement struct {
	q *database.Queries
}

func NewGameManagement(q *database.Queries) (*GameManagement, error) {
	if q == nil {
		return nil, errors.New("database queries are not initialized")
	}
	return &GameManagement{q: q}, nil
}

func (uc *GameManagement) AddGame(ctx context.Context, name string) (model.Game, error) {
	name = strings.TrimSpace(name)

	if len(name) < 3 {
		return model.Game{}, fmt.Errorf(
			"game name cannot have less than 3 characters: '%s'",
			name,
		)
	}

	game, err := uc.q.CreateGame(ctx, name)
	if err != nil {
		return model.Game{}, fmt.Errorf("failed to create game %q: %w", name, err)
	}
	return mapping.ToDomainGame(game), nil
}

func (uc *GameManagement) UpdateGame(ctx context.Context, id int64, name string) error {
	if id < 1 {
		return fmt.Errorf("invalid game ID: %d", id)
	}

	name = strings.TrimSpace(name)

	if len(name) < 3 {
		return fmt.Errorf(
			"new game name cannot have less than 3 characters: '%s'",
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
	if id < 1 {
		return fmt.Errorf("invalid game ID: %d", id)
	}

	if err := uc.q.DeleteGame(ctx, id); err != nil {
		return fmt.Errorf("failed to delete game %d: %w", id, err)
	}
	return nil
}
