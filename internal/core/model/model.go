package model

import "time"

type SharedDataStatus string

const (
	NotStarted SharedDataStatus = "not_started"
	Playing    SharedDataStatus = "playing"
	Paused     SharedDataStatus = "paused"
)

type SharedData struct {
	GameID       int64
	Status       SharedDataStatus
	DeadlineAt   *time.Time
	LastSavedAt  *time.Time
	LastPausedAt *time.Time
}
