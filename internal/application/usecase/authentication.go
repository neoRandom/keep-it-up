package usecase

import (
	"context"
	"errors"
	"fmt"
	"keep-it-up/internal/core/model"
	"keep-it-up/internal/core/port"
	"keep-it-up/internal/core/service"
	"keep-it-up/internal/infrastructure/database"
	"keep-it-up/internal/infrastructure/database/mapping"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUnauthorized = errors.New("player not authorized")
	ErrBadRequest   = errors.New("missing username or password")
)

type Authentication struct {
	q  *database.Queries
	tg port.TokenGenerator
}

func NewAuthentication(q *database.Queries, tg port.TokenGenerator) *Authentication {
	return &Authentication{
		q: q, tg: tg,
	}
}

func (uc *Authentication) CheckPlayerPassword(
	ctx context.Context, username string, password string,
) (model.Player, error) {
	if uc.q == nil {
		return model.Player{}, fmt.Errorf("database queries are not initialized")
	}

	if len(username) < 3 {
		return model.Player{}, fmt.Errorf(
			"username cannot have less than 3 characters: '%s'", username,
		)
	}

	if err := service.IsPasswordValid(password); err != nil {
		return model.Player{}, err
	}

	player, err := uc.q.GetPlayerByUsername(ctx, username)
	if err != nil {
		return model.Player{}, fmt.Errorf("failed to get player by username %q: %w", username, err)
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(player.HashedPassword), []byte(password),
	); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return model.Player{}, nil
		}
		return model.Player{}, fmt.Errorf("failed to compare password hash: %w", err)
	}

	return mapping.ToDomainPlayer(player), nil
}

func (uc *Authentication) LoginPlayer(
	ctx context.Context, username string, password string,
) (model.AuthResult, error) {
	if uc.q == nil {
		return model.AuthResult{}, fmt.Errorf("database queries are not initialized")
	}

	if uc.tg == nil {
		return model.AuthResult{}, fmt.Errorf("token generator is not initialized")
	}

	if strings.TrimSpace(username) == "" || strings.TrimSpace(password) == "" {
		return model.AuthResult{}, ErrBadRequest
	}

	player, err := uc.CheckPlayerPassword(ctx, username, password)
	if err != nil {
		return model.AuthResult{}, fmt.Errorf(
			"failed to check if password is correct: %w", err,
		)
	}
	if player.ID == 0 {
		return model.AuthResult{}, ErrUnauthorized
	}

	token, err := uc.tg.GenerateToken(player)
	if err != nil {
		return model.AuthResult{}, fmt.Errorf("failed to generate token: %w", err)
	}

	return model.AuthResult{
		Token:  token,
		Player: player,
	}, nil
}
