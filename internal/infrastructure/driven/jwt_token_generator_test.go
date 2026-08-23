package driven

import (
	"errors"
	"testing"
	"time"

	"keep-it-up/internal/application/model"
	"keep-it-up/internal/infrastructure/constant"
	coremodel "keep-it-up/internal/core/model"

	"github.com/golang-jwt/jwt/v5"
)

// fixedTime implements port.TimeProvider for deterministic token generation.
type fixedTime struct{ now time.Time }

func (f *fixedTime) Time() (time.Time, error) { return f.now, nil }

func newFixedTime() *fixedTime {
	return &fixedTime{now: time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)}
}

func parseToken(t *testing.T, raw, secret string) *model.JwtPlayerClaims {
	t.Helper()
	claims := &model.JwtPlayerClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if !token.Valid {
		t.Fatal("token is not valid")
	}
	return claims
}

func TestGenerateToken_ValidClaims(t *testing.T) {
	gen := &JwtTokenGenerator{JwtSecret: "secret", TimeProvider: newFixedTime()}

	raw, err := gen.GenerateToken(coremodel.Player{ID: 42, Username: "neo"})
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	claims := parseToken(t, raw, "secret")
	if claims.UserID != 42 {
		t.Errorf("UserID = %d, want 42", claims.UserID)
	}
	if claims.Username != "neo" {
		t.Errorf("Username = %q, want %q", claims.Username, "neo")
	}
}

func TestGenerateToken_ExpiryMatchesSessionLifetime(t *testing.T) {
	gen := &JwtTokenGenerator{JwtSecret: "secret", TimeProvider: newFixedTime()}

	raw, err := gen.GenerateToken(coremodel.Player{ID: 1, Username: "neo"})
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	claims := parseToken(t, raw, "secret")
	wantExp := newFixedTime().now.Add(constant.SessionLifetime)
	if claims.ExpiresAt == nil {
		t.Fatal("ExpiresAt is nil")
	}
	if !claims.ExpiresAt.Time.Equal(wantExp) {
		t.Errorf("ExpiresAt = %v, want %v", claims.ExpiresAt.Time, wantExp)
	}
}

func TestGenerateToken_RejectsWrongSecret(t *testing.T) {
	gen := &JwtTokenGenerator{JwtSecret: "correct", TimeProvider: newFixedTime()}
	raw, err := gen.GenerateToken(coremodel.Player{ID: 1, Username: "neo"})
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	claims := &model.JwtPlayerClaims{}
	_, err = jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte("wrong"), nil
	})
	if err == nil {
		t.Fatal("expected error when verifying with the wrong secret")
	}
}

func TestGenerateToken_Errors(t *testing.T) {
	t.Run("empty secret", func(t *testing.T) {
		gen := &JwtTokenGenerator{JwtSecret: "  ", TimeProvider: newFixedTime()}
		if _, err := gen.GenerateToken(coremodel.Player{}); err == nil {
			t.Fatal("expected error for empty secret")
		}
	})

	t.Run("nil time provider", func(t *testing.T) {
		gen := &JwtTokenGenerator{JwtSecret: "secret", TimeProvider: nil}
		if _, err := gen.GenerateToken(coremodel.Player{}); err == nil {
			t.Fatal("expected error for nil time provider")
		}
	})
}