package model

import "time"

// JwtPlayerClaims are the claims carried in an authentication token, expressed
// as a framework-free struct. JWT-specific (de)serialization lives in the
// driven adapter that talks to the JWT library.
type JwtPlayerClaims struct {
	ExpiresAt time.Time `json:"exp"`
	UserID    int64     `json:"user_id"`
	Username  string    `json:"username"`
}
