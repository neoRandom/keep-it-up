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
	if err != nil {
		return fmt.Errorf("failed to grant player %d access to game %d: %w", playerId, gameId, err)
	}

	return nil
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

	if err := uc.q.RevokePlayerAccess(
		ctx,
		database.RevokePlayerAccessParams{
			GameID:   gameId,
			PlayerID: playerId,
		},
	); err != nil {
		return fmt.Errorf("failed to revoke player %d access to game %d: %w", playerId, gameId, err)
	}
	return nil
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
	
	granted, err := uc.q.CheckPlayerAccess(
		ctx,
		database.CheckPlayerAccessParams{
			GameID: gameId,
			PlayerID: playerId,
		},
	)
	if err != nil {
		return false, fmt.Errorf("failed to check player %d access to game %d: %w", playerId, gameId, err)
	}
	return granted, nil
}
