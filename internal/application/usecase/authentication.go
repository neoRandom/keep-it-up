package usecase

import (
	"context"
	"errors"
	"fmt"
	"keep-it-up/internal/infrastructure/database"

	"golang.org/x/crypto/bcrypt"
)

type Authentication struct {
	q *database.Queries
}

func NewAuthentication(q *database.Queries) *Authentication {
	return &Authentication{q: q}
}

func (uc *Authentication) GeneratePasswordHash(password string) (string, error) {
	if len(password) < 6 {
		return "", fmt.Errorf("Password cannot have less than 6 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

func (uc *Authentication) CheckPlayerPassword(ctx context.Context, username string, password string) (bool, error) {
	if uc.q == nil {
		return false, fmt.Errorf("database queries are not initialized")
	}

	if len(username) < 3 {
		return false, fmt.Errorf("Username cannot have less than 3 characters: '%s'", username)
	}

	if len(password) < 6 {
		return false, fmt.Errorf("Password cannot have less than 6 characters")
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

func (uc *Authentication) LoginPlayer() {}
