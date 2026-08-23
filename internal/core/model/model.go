package model

import "time"

type SharedDataStatus string

const (
	NotStarted SharedDataStatus = "not_started"
	Playing    SharedDataStatus = "playing"
	Paused     SharedDataStatus = "paused"
)

// Game is the domain representation of a playable game.
type Game struct {
	ID   int64
	Name string
}

// Player is a system account. HashedPassword is deliberately excluded: it is a
// persistence detail consumed only inside the authentication use case from the
// DB-fetched value, and is never part of the domain vocabulary exposed via ports.
type Player struct {
	ID       int64
	Name     string
	Username string
}

// Interaction is a single state-machine event on a game. PlayerID and SavedBy
// are optional (nil), mirroring the DB columns using idiomatic pointers.
type Interaction struct {
	ID         int64
	GameID     int64
	PlayerID   *int64
	Action     string
	OccurredAt time.Time
	SavedBy    *int64
}

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
	Player Player
}
