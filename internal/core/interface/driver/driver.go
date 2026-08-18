package driver

import (
	"context"
	"keep-it-up/internal/core/model"
	"keep-it-up/internal/infrastructure/database"
	"time"
)

type GameManagement interface {
	AddGame(ctx context.Context, name string) (database.Game, error)
	UpdateGame(ctx context.Context, id int64, name string) error
	DeleteGame(ctx context.Context, id int64) error
}

type AccessManagement interface {
	GrantPlayerAccess(ctx context.Context, gameId int64, playerId int64) error
	RevokePlayerAccess(ctx context.Context, gameId int64, playerId int64) error
}

type PlayerManagement interface {
	AddPlayer(ctx context.Context, name string, username string, password string) (database.Player, error)
	UpdatePlayerName(ctx context.Context, id int64, name string) error
	UpdatePlayerPassword(ctx context.Context, id int64, currentPassword string, newPassword string) error
	UpdatePlayerPasswordForce(ctx context.Context, id int64, password string) error
	DeletePlayer(ctx context.Context, id int64) error
}

type Authentication interface {
	IsPasswordValid(password string) error
	GeneratePasswordHash(password string) (string, error)
	CheckPlayerPassword(ctx context.Context, username string, password string) (bool, error)
	LoginPlayer(ctx context.Context, username string, password string) (model.AuthResult, error)
}

type DataFetching interface {
	ListPlayerGames(ctx context.Context, playerId int64) ([]database.Game, error)
	GetSharedData(ctx context.Context, gameId int64) (*model.SharedData, error)
	ListInteractions(ctx context.Context, gameId int64, limit int64) ([]database.Interaction, error)
}

type GameCommands interface {
	SaveGame(ctx context.Context, gameId int64, playerId int64, amount time.Time) error
	ResumeGame(ctx context.Context, gameId int64, playerId int64) error
	PauseGame(ctx context.Context, gameId int64, playerId int64) error
}
