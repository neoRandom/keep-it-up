package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"keep-it-up/internal/infrastructure/constant"
	"keep-it-up/internal/infrastructure/util"
)

// Config holds the runtime configuration for the server, loaded from the
// environment (and the dotenv file, if present).
type Config struct {
	JWTSecret         string
	ServerAddress     string
	DBString          string
	ValkeyAddress     string
	IdempotencyTTL    time.Duration
	IdempotencyHeader string
}

// Load reads the dotenv file and the process environment, resolving the values
// the server needs to start. It fails fast if any required variable is missing
// or cannot be resolved, so a misconfigured server never starts half-initialized.
func Load() (Config, error) {
	util.LoadEnv(constant.EnvFilename)

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return Config{}, fmt.Errorf("missing required environment variable JWT_SECRET")
	}

	addr := os.Getenv("SERVER_ADDRESS")
	if addr == "" {
		return Config{}, fmt.Errorf("missing required environment variable SERVER_ADDRESS")
	}

	dbString, err := filepath.Abs(os.Getenv("GOOSE_DBSTRING"))
	if err != nil {
		return Config{}, fmt.Errorf("failed to get dbstring absolute path: %w", err)
	}

	valkeyAddress := os.Getenv("VALKEY_ADDRESS")
	if valkeyAddress == "" {
		return Config{}, fmt.Errorf("missing required environment variable VALKEY_ADDRESS")
	}

	idempotencyTTL, err := time.ParseDuration(os.Getenv("IDEMPOTENCY_TTL"))
	if err != nil {
		return Config{}, fmt.Errorf("invalid IDEMPOTENCY_TTL: %w", err)
	}
	if idempotencyTTL <= 0 {
		return Config{}, fmt.Errorf("IDEMPOTENCY_TTL must be a positive duration")
	}

	idempotencyHeader := os.Getenv("IDEMPOTENCY_HEADER")
	if idempotencyHeader == "" {
		return Config{}, fmt.Errorf("missing required environment variable IDEMPOTENCY_HEADER")
	}

	return Config{
		JWTSecret:         jwtSecret,
		ServerAddress:     addr,
		DBString:          dbString,
		ValkeyAddress:     valkeyAddress,
		IdempotencyTTL:    idempotencyTTL,
		IdempotencyHeader: idempotencyHeader,
	}, nil
}
