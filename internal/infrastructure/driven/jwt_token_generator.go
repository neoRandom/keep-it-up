package driven

import (
	"errors"
	"fmt"
	"keep-it-up/internal/application/model"
	"keep-it-up/internal/core/port"
	"keep-it-up/internal/infrastructure/constant"
	coremodel "keep-it-up/internal/core/model"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type JwtTokenGenerator struct {
	JwtSecret    string
	TimeProvider port.TimeProvider
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
	
	claims := &model.JwtPlayerClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(
				t.Add(constant.SessionLifetime),
			),
		},
		UserID: player.ID,
		Username: player.Username,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(g.JwtSecret))
}
