package usecase

import "context"

type AccessManagement struct{}

func NewAccessManagement() *AccessManagement {
	return &AccessManagement{}
}

func (uc *AccessManagement) GivePlayerAccess(
	ctx context.Context, gameId int64, playerId int64,
) error {
	return nil
}

func (uc *AccessManagement) RemovePlayerAccess(
	ctx context.Context, gameId int64, playerId int64,
) error {
	return nil
}
