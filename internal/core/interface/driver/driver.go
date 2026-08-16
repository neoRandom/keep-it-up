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
	AddPlayer()
	UpdatePlayer()
	DeletePlayer()
}

type Authentication interface {
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
