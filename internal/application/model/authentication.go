package model

import "github.com/golang-jwt/jwt/v4"

type JwtPlayerClaims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}
