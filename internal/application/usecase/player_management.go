package usecase

import (
	"context"
	"fmt"
	"keep-it-up/internal/infrastructure/database"
	"keep-it-up/internal/infrastructure/util"
)

type PlayerManagement struct {
	q *database.Queries
}

func NewPlayerManagement(q *database.Queries) *PlayerManagement {
	return &PlayerManagement{q: q}
}

func (uc *PlayerManagement) AddPlayer(ctx context.Context, name string, username string, password string) (database.Player, error) {
	if uc.q == nil {
		return database.Player{}, fmt.Errorf("database queries are not initialized")
	}

	if len(name) < 2 {
		return database.Player{}, fmt.Errorf("Player name cannot have less than 2 characters: '%s'", name)
	}

	if !util.IsAlphanumeric(name) {
		return database.Player{}, fmt.Errorf("Player name isn't purely alphanumeric: '%s'", name)
	}

	if len(username) < 3 {
		return database.Player{}, fmt.Errorf("Username cannot have less than 3 characters: '%s'", username)
	}

	if !util.IsAlphanumeric(username) {
		return database.Player{}, fmt.Errorf("Username isn't purely alphanumeric: '%s'", username)
	}

	if len(password) < 6 {
		return database.Player{}, fmt.Errorf("Password cannot have less than 6 characters")
	}

	// TODO: generate a password hash before passing a password to a query that expects a hashed password.
	return uc.q.CreatePlayer(ctx, database.CreatePlayerParams{
		Name:           name,
		Username:       username,
		HashedPassword: password,
	})
}

func (uc *PlayerManagement) UpdatePlayerName(ctx context.Context, id int64, name string) error {
	if uc.q == nil {
		return fmt.Errorf("database queries are not initialized")
	}

	if id < 1 {
		return fmt.Errorf("Invalid player ID: %d", id)
	}

	if len(name) < 2 {
		return fmt.Errorf("Player name cannot have less than 2 characters: '%s'", name)
	}

	if !util.IsAlphanumeric(name) {
		return fmt.Errorf("Player name isn't purely alphanumeric: '%s'", name)
	}

	return uc.q.UpdatePlayerName(ctx, database.UpdatePlayerNameParams{
		ID:   id,
		Name: name,
	})
}

func (uc *PlayerManagement) UpdatePlayerPassword(ctx context.Context, id int64, password string) error {
	if uc.q == nil {
		return fmt.Errorf("database queries are not initialized")
	}

	if id < 1 {
		return fmt.Errorf("Invalid player ID: %d", id)
	}

	if len(password) < 6 {
		return fmt.Errorf("Password cannot have less than 6 characters")
	}

	// TODO: generate a password hash before passing a password to a query that expects a hashed password.
	return uc.q.UpdatePlayerPassword(ctx, database.UpdatePlayerPasswordParams{
		ID:             id,
		HashedPassword: password,
	})
}

func (uc *PlayerManagement) DeletePlayer(ctx context.Context, id int64) error {
	if uc.q == nil {
		return fmt.Errorf("database queries are not initialized")
	}

	if id < 1 {
		return fmt.Errorf("Invalid player ID: %d", id)
	}

	return uc.q.DeletePlayer(ctx, id)
}


