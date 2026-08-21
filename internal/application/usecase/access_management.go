package usecase

import (
	"context"
	"errors"
	"fmt"
	"keep-it-up/internal/infrastructure/database"
)

type AccessManagement struct {
	q *database.Queries
}

func NewAccessManagement(q *database.Queries) *AccessManagement {
	return &AccessManagement{q: q}
}

func (uc *AccessManagement) GrantPlayerAccess(
	ctx context.Context, gameId int64, playerId int64,
) error {
	if uc.q == nil {
		return errors.New("database queries are not initialized")
	}

	if gameId < 1 {
		return fmt.Errorf("invalid game ID: %d", gameId)
	}

	if playerId < 1 {
		return fmt.Errorf("invalid player ID: %d", playerId)
	}

	_, err := uc.q.GrantPlayerAccess(
		ctx,
		database.GrantPlayerAccessParams{
			GameID:   gameId,
			PlayerID: playerId,
		},
	)

	return err
}

func (uc *AccessManagement) RevokePlayerAccess(
	ctx context.Context, gameId int64, playerId int64,
) error {
	if uc.q == nil {
		return errors.New("database queries are not initialized")
	}

	if gameId < 1 {
		return fmt.Errorf("invalid game ID: %d", gameId)
	}

	if playerId < 1 {
		return fmt.Errorf("invalid player ID: %d", playerId)
	}

	return uc.q.RevokePlayerAccess(
		ctx,
		database.RevokePlayerAccessParams{
			GameID:   gameId,
			PlayerID: playerId,
		},
	)
}

func (uc *AccessManagement) CheckPlayerAccess(
	ctx context.Context, gameId, playerId int64,
) (bool, error) {
	if uc.q == nil {
		return false, errors.New("database queries are not initialized")
	}

	if gameId < 1 {
		return false, fmt.Errorf("invalid game ID: %d", gameId)
	}

	if playerId < 1 {
		return false, fmt.Errorf("invalid player ID: %d", playerId)
	}
	
	return uc.q.CheckPlayerAccess(
		ctx,
		database.CheckPlayerAccessParams{
			GameID: gameId,
			PlayerID: playerId,
		},
	)
}
