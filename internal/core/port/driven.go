package port

import (
	"keep-it-up/internal/core/model"
	"time"
)

type TimeProvider interface {
	Time() (time.Time, error)
}

type TokenGenerator interface {
	GenerateToken(model.Player) (string, error)
}
