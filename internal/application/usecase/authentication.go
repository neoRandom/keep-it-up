package usecase

import (
	"context"
	"errors"
	"fmt"
	"keep-it-up/internal/core/model"
	"keep-it-up/internal/core/service"
	"keep-it-up/internal/infrastructure/database"

	"golang.org/x/crypto/bcrypt"
)

type Authentication struct {
	q *database.Queries
}

func NewAuthentication(q *database.Queries) *Authentication {
	return &Authentication{q: q}
}

func (uc *Authentication) CheckPlayerPassword(ctx context.Context, username string, password string) (bool, error) {
	if uc.q == nil {
		return false, fmt.Errorf("database queries are not initialized")
	}

	if len(username) < 3 {
		return false, fmt.Errorf("Username cannot have less than 3 characters: '%s'", username)
	}

	if err := service.IsPasswordValid(password); err != nil {
		return false, err
	}

	player, err := uc.q.GetPlayerByUsername(ctx, username)
	if err != nil {
		return false, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(player.HashedPassword), []byte(password)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (uc *Authentication) LoginPlayer(
	ctx context.Context, username string, password string,
) (model.AuthResult, error) {
	// TODO: Needed for web-based authentication
	return model.AuthResult{}, nil
}
