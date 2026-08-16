package driver

import (
	"context"
	"keep-it-up/internal/infrastructure/database"
)

type GameManagement interface {
	AddGame(ctx context.Context, name string) (database.Game, error)
	UpdateGame(ctx context.Context, id int64, name string) error
	DeleteGame(ctx context.Context, id int64) error
}

type PlayerManagement interface {
	AddPlayer(ctx context.Context, name string, username string, password string) (database.Player, error)
	UpdatePlayerName(ctx context.Context, id int64, name string) error
	UpdatePlayerPassword(ctx context.Context, id int64, currentPassword string, newPassword string) error
	UpdatePlayerPasswordForce(ctx context.Context, id int64, password string) error
	DeletePlayer(ctx context.Context, id int64) error
}

type Authentication interface {
	GeneratePasswordHash(password string) (string, error)
	VerifyPlayerPassword(password string) error
	CheckPlayerPassword(ctx context.Context, username string, password string) (bool, error)
	LoginPlayer()
}

type DataFetching interface {
	ListPlayerGames()
	GetSharedData()
	ListInteractions()
}

type GameCommands interface {
	SaveGame()
	ResumeGame()
	PauseGame()
}
