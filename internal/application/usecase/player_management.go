package usecase

import (
	"context"
	"errors"
	"fmt"
	"keep-it-up/internal/core/port"
	"keep-it-up/internal/core/service"
	"keep-it-up/internal/infrastructure/database"
	"keep-it-up/internal/core/model"
	"keep-it-up/internal/infrastructure/database/mapping"
	"keep-it-up/internal/infrastructure/util"
	"strings"
)

type PlayerManagement struct {
	q    *database.Queries
	auth port.Authentication
}

func NewPlayerManagement(q *database.Queries, auth port.Authentication) (*PlayerManagement, error) {
	if auth == nil {
		return nil, errors.New("authentication is not initialized")
	}
	return &PlayerManagement{q: q, auth: auth}, nil
}

func (uc *PlayerManagement) AddPlayer(ctx context.Context, name string, username string, password string) (model.Player, error) {
	if uc.q == nil {
		return model.Player{}, fmt.Errorf("database queries are not initialized")
	}
	if uc.auth == nil {
		return model.Player{}, fmt.Errorf("authentication is not initialized")
	}

	name = strings.TrimSpace(name)

	if len(name) < 2 {
		return model.Player{}, fmt.Errorf(
			"player name cannot have less than 2 characters: '%s'",
			name,
		)
	}

	username = strings.TrimSpace(username)

	if len(username) < 3 {
		return model.Player{}, fmt.Errorf(
			"username cannot have less than 3 characters: '%s'",
			username,
		)
	}

	if !util.IsAlphanumeric(username) {
		return model.Player{}, fmt.Errorf(
			"username isn't purely alphanumeric: '%s'",
			username,
		)
	}

	password = strings.TrimSpace(password)

	if password == username {
		return model.Player{}, errors.New("player password cannot be equal to its username")
	}

	if err := service.IsPasswordValid(password); err != nil {
		return model.Player{}, err
	}

	hashedPassword, err := service.GeneratePasswordHash(password)
	if err != nil {
		return model.Player{}, fmt.Errorf("failed to generate password hash: %w", err)
	}

	player, err := uc.q.CreatePlayer(ctx, database.CreatePlayerParams{
		Name:           name,
		Username:       username,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		return model.Player{}, fmt.Errorf("failed to create player %q: %w", username, err)
	}
	return mapping.ToDomainPlayer(player), nil
}

func (uc *PlayerManagement) UpdatePlayerName(ctx context.Context, playerId int64, name string) error {
	if uc.q == nil {
		return fmt.Errorf("database queries are not initialized")
	}

	if playerId < 1 {
		return fmt.Errorf("invalid player ID: %d", playerId)
	}

	name = strings.TrimSpace(name)

	if len(name) < 2 {
		return fmt.Errorf("player name cannot have less than 2 characters: '%s'", name)
	}

	if err := uc.q.UpdatePlayerName(ctx, database.UpdatePlayerNameParams{
		ID:   playerId,
		Name: name,
	}); err != nil {
		return fmt.Errorf("failed to update player name for id %d: %w", playerId, err)
	}
	return nil
}

func (uc *PlayerManagement) BaseUpdatePlayerPassword(ctx context.Context, id int64, password string) error {
	if uc.q == nil {
		return fmt.Errorf("database queries are not initialized")
	}
	if uc.auth == nil {
		return fmt.Errorf("authentication is not initialized")
	}

	if id < 1 {
		return fmt.Errorf("invalid player ID: %d", id)
	}

	password = strings.TrimSpace(password)

	if err := service.IsPasswordValid(password); err != nil {
		return err
	}

	hashedPassword, err := service.GeneratePasswordHash(password)
	if err != nil {
		return fmt.Errorf("failed to generate password hash: %w", err)
	}

	if err := uc.q.UpdatePlayerPassword(ctx, database.UpdatePlayerPasswordParams{
		ID:             id,
		HashedPassword: hashedPassword,
	}); err != nil {
		return fmt.Errorf("failed to update password for player %d: %w", id, err)
	}
	return nil
}

func (uc *PlayerManagement) UpdatePlayerPassword(
	ctx context.Context, username, currentPassword, newPassword string,
) error {
	if uc.q == nil {
		return errors.New("database queries are not initialized")
	}
	if uc.auth == nil {
		return errors.New("authentication is not initialized")
	}

	if strings.TrimSpace(username) == "" {
		return errors.New("username cannot be empty string")
	}

	currentPassword = strings.TrimSpace(currentPassword)
	if err := service.IsPasswordValid(currentPassword); err != nil {
		return err
	}

	newPassword = strings.TrimSpace(newPassword)
	if err := service.IsPasswordValid(newPassword); err != nil {
		return err
	}

	player, err := uc.auth.CheckPlayerPassword(ctx, username, currentPassword)
	if err != nil {
		return err
	}
	if player.ID == 0 {
		return errors.New("incorrect current password")
	}
	
	if newPassword == player.Username {
		return errors.New("new player password cannot be equal to its username")
	}

	return uc.BaseUpdatePlayerPassword(ctx, player.ID, newPassword)
}

func (uc *PlayerManagement) UpdatePlayerPasswordForce(
	ctx context.Context, username, password string,
) error {
	if uc.q == nil {
		return errors.New("database queries are not initialized")
	}
	if uc.auth == nil {
		return errors.New("authentication is not initialized")
	}

	if strings.TrimSpace(username) == "" {
		return errors.New("username cannot be empty string")
	}

	if err := service.IsPasswordValid(password); err != nil {
		return err
	}
	
	player, err := uc.q.GetPlayerByUsername(ctx, username)
	if err != nil {
		return fmt.Errorf("failed to get player by username: %w", err)
	}
	if player.ID == 0 {
		return errors.New("player does not exist")
	}

	return uc.BaseUpdatePlayerPassword(ctx, player.ID, password)
}

func (uc *PlayerManagement) DeletePlayer(ctx context.Context, playerId int64) error {
	if uc.q == nil {
		return fmt.Errorf("database queries are not initialized")
	}

	if playerId < 1 {
		return fmt.Errorf("invalid player ID: %d", playerId)
	}

	if err := uc.q.DeletePlayer(ctx, playerId); err != nil {
		return fmt.Errorf("failed to delete player %d: %w", playerId, err)
	}
	return nil
}
