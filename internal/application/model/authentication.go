package model

import "github.com/golang-jwt/jwt/v5"

type JwtPlayerClaims struct {
	jwt.RegisteredClaims
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
}
