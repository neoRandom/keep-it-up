package usecase

import (
	"context"
	"fmt"
	"keep-it-up/internal/core/interface/driver"
	"keep-it-up/internal/infrastructure/database"
	"keep-it-up/internal/infrastructure/util"
	"strings"
)

type PlayerManagement struct {
	q    *database.Queries
	auth driver.Authentication
}

func NewPlayerManagement(q *database.Queries, auth driver.Authentication) *PlayerManagement {
	if auth == nil {
		return nil
	}
	return &PlayerManagement{q: q, auth: auth}
}

func (uc *PlayerManagement) AddPlayer(ctx context.Context, name string, username string, password string) (database.Player, error) {
	if uc.q == nil {
		return database.Player{}, fmt.Errorf("database queries are not initialized")
	}
	if uc.auth == nil {
		return database.Player{}, fmt.Errorf("authentication is not initialized")
	}

	name = strings.TrimSpace(name)
	
	if len(name) < 2 {
		return database.Player{}, fmt.Errorf(
			"Player name cannot have less than 2 characters: '%s'", 
			name,
		)
	}

	if !util.IsAlphanumeric(name) {
		return database.Player{}, fmt.Errorf(
			"Player name isn't purely alphanumeric: '%s'", 
			name,
		)
	}

	username = strings.TrimSpace(username)
	
	if len(username) < 3 {
		return database.Player{}, fmt.Errorf(
			"Username cannot have less than 3 characters: '%s'", 
			username,
		)
	}
	
	if !util.IsAlphanumeric(username) {
		return database.Player{}, fmt.Errorf(
			"Username isn't purely alphanumeric: '%s'", 
			username,
		)
	}
	
	password = strings.TrimSpace(password)

	if err := uc.auth.VerifyPlayerPassword(password); err != nil {
		return database.Player{}, err
	}

	hashedPassword, err := uc.auth.GeneratePasswordHash(password)
	if err != nil {
		return database.Player{}, err
	}

	return uc.q.CreatePlayer(ctx, database.CreatePlayerParams{
		Name:           name,
		Username:       username,
		HashedPassword: hashedPassword,
	})
}

func (uc *PlayerManagement) UpdatePlayerName(ctx context.Context, id int64, name string) error {
	if uc.q == nil {
		return fmt.Errorf("database queries are not initialized")
	}

	if id < 1 {
		return fmt.Errorf("Invalid player ID: %d", id)
	}
	
	name = strings.TrimSpace(name)

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

func (uc *PlayerManagement) BaseUpdatePlayerPassword(ctx context.Context, id int64, password string) error {
	if uc.q == nil {
		return fmt.Errorf("database queries are not initialized")
	}
	if uc.auth == nil {
		return fmt.Errorf("authentication is not initialized")
	}

	if id < 1 {
		return fmt.Errorf("Invalid player ID: %d", id)
	}
	
	password = strings.TrimSpace(password)

	if err := uc.auth.VerifyPlayerPassword(password); err != nil {
		return err
	}

	hashedPassword, err := uc.auth.GeneratePasswordHash(password)
	if err != nil {
		return err
	}

	return uc.q.UpdatePlayerPassword(ctx, database.UpdatePlayerPasswordParams{
		ID:             id,
		HashedPassword: hashedPassword,
	})
}

func (uc *PlayerManagement) UpdatePlayerPassword(ctx context.Context, id int64, currentPassword string, newPassword string) error {
	if uc.q == nil {
		return fmt.Errorf("database queries are not initialized")
	}
	if uc.auth == nil {
		return fmt.Errorf("authentication is not initialized")
	}

	if id < 1 {
		return fmt.Errorf("Invalid player ID: %d", id)
	}

	player, err := uc.q.GetPlayer(ctx, id)
	if err != nil {
		return err
	}

	currentPassword = strings.TrimSpace(currentPassword)
	if err := uc.auth.VerifyPlayerPassword(currentPassword); err != nil {
		return err
	}
	
	newPassword = strings.TrimSpace(newPassword)
	if err := uc.auth.VerifyPlayerPassword(newPassword); err != nil {
		return err
	}

	valid, err := uc.auth.CheckPlayerPassword(ctx, player.Username, currentPassword)
	if err != nil {
		return err
	}
	if !valid {
		return fmt.Errorf("incorrect current password")
	}

	return uc.BaseUpdatePlayerPassword(ctx, id, newPassword)
}

func (uc *PlayerManagement) UpdatePlayerPasswordForce(ctx context.Context, id int64, password string) error {
	if uc.q == nil {
		return fmt.Errorf("database queries are not initialized")
	}
	if uc.auth == nil {
		return fmt.Errorf("authentication is not initialized")
	}

	if id < 1 {
		return fmt.Errorf("Invalid player ID: %d", id)
	}

	if err := uc.auth.VerifyPlayerPassword(password); err != nil {
		return err
	}

	return uc.BaseUpdatePlayerPassword(ctx, id, password)
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
