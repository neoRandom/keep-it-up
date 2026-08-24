package driven

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"keep-it-up/internal/application/model"
	coremodel "keep-it-up/internal/core/model"
	"keep-it-up/internal/core/port"

	"github.com/golang-jwt/jwt/v5"
)

type JwtTokenGenerator struct {
	JwtSecret       string
	TimeProvider    port.TimeProvider
	SessionLifetime time.Duration
}

// JwtClaims is the JWT-library-specific claims representation used for signing
// and parsing tokens. It embeds the library registered claims and carries the
// app player fields; it mirrors application/model.JwtPlayerClaims (the
// framework-free DTO) at the JWT boundary.
type JwtClaims struct {
	jwt.RegisteredClaims
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
}

func (g *JwtTokenGenerator) GenerateToken(player coremodel.Player) (string, error) {
	if strings.TrimSpace(g.JwtSecret) == "" {
		return "", errors.New("jwt secret cannot be empty string")
	}
	if g.TimeProvider == nil {
		return "", errors.New("time provider is not initialized")
	}

	t, err := g.TimeProvider.Time()
	if err != nil {
		return "", fmt.Errorf("failed to get current time: %w", err)
	}

	claims := &JwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(t.Add(g.SessionLifetime)),
		},
		UserID:   player.ID,
		Username: player.Username,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(g.JwtSecret))
}

// ParseToken deserializes a raw token into the framework-free application
// claims DTO, keeping all JWT-library interaction inside the driven adapter.
func (g *JwtTokenGenerator) ParseToken(raw, secret string) (model.JwtPlayerClaims, error) {
	claims := &JwtClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return model.JwtPlayerClaims{}, err
	}
	if !token.Valid {
		return model.JwtPlayerClaims{}, errors.New("token is not valid")
	}
	result := model.JwtPlayerClaims{UserID: claims.UserID, Username: claims.Username}
	if claims.ExpiresAt != nil {
		result.ExpiresAt = claims.ExpiresAt.Time
	}
	return result, nil
}
