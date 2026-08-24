package port

import (
	"context"
	"keep-it-up/internal/core/model"
)

type GameManagement interface {
	AddGame(ctx context.Context, name string) (model.Game, error)
	UpdateGame(ctx context.Context, id int64, name string) error
	DeleteGame(ctx context.Context, id int64) error
}

type AccessManagement interface {
	GrantPlayerAccess(ctx context.Context, gameId, playerId int64) error
	RevokePlayerAccess(ctx context.Context, gameId, playerId int64) error
	CheckPlayerAccess(ctx context.Context, gameId, playerId int64) (bool, error)
}

type PlayerManagement interface {
	AddPlayer(
		ctx context.Context, name, username, password string,
	) (model.Player, error)
	UpdatePlayerName(
		ctx context.Context, playerId int64, name string,
	) error
	UpdatePlayerPassword(
		ctx context.Context, username, currentPassword, newPassword string,
	) error
	UpdatePlayerPasswordForce(
		ctx context.Context, username, password string,
	) error
	DeletePlayer(
		ctx context.Context, playerId int64,
	) error
}

type Authentication interface {
	CheckPlayerPassword(
		ctx context.Context, username, password string,
	) (model.Player, error)
	LoginPlayer(
		ctx context.Context, username, password string,
	) (model.AuthResult, error)
}

type DataFetching interface {
	ListPlayerGames(
		ctx context.Context, playerId int64,
	) ([]model.Game, error)
	GetSharedData(
		ctx context.Context, gameId int64,
	) (*model.SharedData, error)
	ListInteractions(
		ctx context.Context, gameId, limit, offset int64,
	) ([]model.Interaction, error)
	ListPlayerInteractions(
		ctx context.Context, gameId, playerId, limit, offset int64,
	) ([]model.Interaction, error)
	FirstInteraction(
		ctx context.Context, gameId int64,
	) (*model.Interaction, error)
	LastInteraction(
		ctx context.Context, gameId int64,
	) (*model.Interaction, error)
}

type GameCommands interface {
	SaveGame(ctx context.Context, gameId, playerId, duration int64) error
	ResumeGame(ctx context.Context, gameId, playerId int64) error
	PauseGame(ctx context.Context, gameId, playerId int64) error
}
