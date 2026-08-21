package port

import (
	"keep-it-up/internal/infrastructure/database"
	"time"
)

type TimeProvider interface {
	Time() (time.Time, error)
}

type TokenGenerator interface {
	GenerateToken(database.Player) (string, error)
}
