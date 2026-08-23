package httpadapter

import (
	"database/sql"
	"time"
)

// SessionCookieName matches the `sessionCookie` security scheme in api/openapi.yaml.
const SessionCookieName string = "session"

// defaultInteractionsLimit is used when the `limit` query param is omitted.
const defaultInteractionsLimit int64 = 20

// defaultInteractionsOffset is used when the `offset` query param is omitted.
const defaultInteractionsOffset int64 = 0

// Interactions query param enum values. `query=all` is the legacy behavior.
const (
	queryAll    = "all"
	queryPlayer = "player"
	queryFirst  = "first"
	queryLast   = "last"
)

// Client-facing error messages. Kept as constants so wording stays consistent
// across handlers.
const (
	msgAuthenticationRequired = "Authentication required"
	msgInvalidGameID          = "Invalid gameId"
	msgGameNotFound           = "Game not found or inaccessible"
	msgGameNotPlaying         = "Game is not currently playing"
	msgGameNotPaused          = "Game is not currently paused"
)

type gameDTO struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type sharedDataDTO struct {
	GameID       int64      `json:"gameId"`
	Status       string     `json:"status"`
	Valid        *bool      `json:"valid"`
	DeadlineAt   *time.Time `json:"deadlineAt"`
	LastSavedAt  *time.Time `json:"lastSavedAt"`
	LastPausedAt *time.Time `json:"lastPausedAt"`
}

type interactionDTO struct {
	ID         int64  `json:"id"`
	GameID     int64  `json:"gameId"`
	PlayerID   *int64 `json:"playerId"`
	Action     string `json:"action"`
	OccurredAt string `json:"occurredAt"`
	SavedBy    *int64 `json:"savedBy"`
}

type saveRequest struct {
	Duration int64 `json:"duration"`
}

// nullableInt64 maps a database.NullInt64 to *int64, nil when invalid, so
// nullable IDs serialize as JSON null instead of an object.
func nullableInt64(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	x := v.Int64
	return &x
}