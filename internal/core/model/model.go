package model

import (
	"keep-it-up/internal/infrastructure/database"
	"time"
)

type SharedDataStatus string

const (
	NotStarted SharedDataStatus = "not_started"
	Playing    SharedDataStatus = "playing"
	Paused     SharedDataStatus = "paused"
)

type SharedData struct {
	GameID       int64
	Valid        *bool
	Status       SharedDataStatus
	DeadlineAt   *time.Time
	LastSavedAt  *time.Time
	LastPausedAt *time.Time
}

type AuthResult struct {
	Token  string
	Player database.Player
}
