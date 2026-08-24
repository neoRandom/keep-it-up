package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// idempotencyEnvNames are the env vars introduced by idempotency support. They
// are restored around tests that mutate the environment.
var idempotencyEnvNames = []string{"VALKEY_ADDRESS", "IDEMPOTENCY_TTL", "IDEMPOTENCY_HEADER"}

// setRequiredEnv sets every required variable so individual missing-variable
// tests can unset exactly one. Callers must defer restoreEnv.
func setRequiredEnv() map[string]string {
	old := map[string]string{}
	for _, name := range append([]string{"JWT_SECRET", "SERVER_ADDRESS", "GOOSE_DBSTRING"}, idempotencyEnvNames...) {
		old[name] = os.Getenv(name)
	}
	_ = os.Setenv("JWT_SECRET", "secret")
	_ = os.Setenv("SERVER_ADDRESS", ":8080")
	_ = os.Setenv("GOOSE_DBSTRING", "db.sqlite")
	_ = os.Setenv("VALKEY_ADDRESS", "localhost:6379")
	_ = os.Setenv("IDEMPOTENCY_TTL", "24h")
	_ = os.Setenv("IDEMPOTENCY_HEADER", "Idempotency-Key")
	return old
}

func restoreEnv(old map[string]string) {
	for name, val := range old {
		_ = os.Setenv(name, val)
	}
}

func TestLoad(t *testing.T) {
	// LoadEnv reads ".env" if present; ensure a missing file doesn't break the test.
	old := setRequiredEnv()
	defer restoreEnv(old)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.JWTSecret != "secret" {
		t.Errorf("JWTSecret = %q, want %q", cfg.JWTSecret, "secret")
	}
	if cfg.ServerAddress != ":8080" {
		t.Errorf("ServerAddress = %q, want %q", cfg.ServerAddress, ":8080")
	}
	if !filepath.IsAbs(cfg.DBString) {
		t.Errorf("DBString = %q, want absolute path", cfg.DBString)
	}
	wantDB := filepath.Join(mustGetwd(t), "db.sqlite")
	if cfg.DBString != wantDB {
		t.Errorf("DBString = %q, want %q", cfg.DBString, wantDB)
	}
	if cfg.ValkeyAddress != "localhost:6379" {
		t.Errorf("ValkeyAddress = %q, want %q", cfg.ValkeyAddress, "localhost:6379")
	}
	if cfg.IdempotencyTTL != 24*time.Hour {
		t.Errorf("IdempotencyTTL = %v, want %v", cfg.IdempotencyTTL, 24*time.Hour)
	}
	if cfg.IdempotencyHeader != "Idempotency-Key" {
		t.Errorf("IdempotencyHeader = %q, want %q", cfg.IdempotencyHeader, "Idempotency-Key")
	}
	if cfg.SessionLifetime != 72*time.Hour {
		t.Errorf("SessionLifetime = %v, want %v", cfg.SessionLifetime, 72*time.Hour)
	}
	if cfg.InteractionsLimit != 20 {
		t.Errorf("InteractionsLimit = %d, want 20", cfg.InteractionsLimit)
	}
}

func TestLoadMissingValkeyAddress(t *testing.T) {
	old := setRequiredEnv()
	defer restoreEnv(old)
	_ = os.Setenv("VALKEY_ADDRESS", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for missing VALKEY_ADDRESS, got nil")
	}
	if !strings.Contains(err.Error(), "VALKEY_ADDRESS") {
		t.Errorf("error = %q, want mention of VALKEY_ADDRESS", err)
	}
}

func TestLoadInvalidIdempotencyTTL(t *testing.T) {
	old := setRequiredEnv()
	defer restoreEnv(old)
	_ = os.Setenv("IDEMPOTENCY_TTL", "not-a-duration")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for invalid IDEMPOTENCY_TTL, got nil")
	}
	if !strings.Contains(err.Error(), "IDEMPOTENCY_TTL") {
		t.Errorf("error = %q, want mention of IDEMPOTENCY_TTL", err)
	}
}

func TestLoadSubSecondIdempotencyTTL(t *testing.T) {
	old := setRequiredEnv()
	defer restoreEnv(old)
	_ = os.Setenv("IDEMPOTENCY_TTL", "500ms")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for sub-second IDEMPOTENCY_TTL, got nil")
	}
	if !strings.Contains(err.Error(), "IDEMPOTENCY_TTL") {
		t.Errorf("error = %q, want mention of IDEMPOTENCY_TTL", err)
	}
}

func TestLoadMissingIdempotencyHeader(t *testing.T) {
	old := setRequiredEnv()
	defer restoreEnv(old)
	_ = os.Setenv("IDEMPOTENCY_HEADER", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for missing IDEMPOTENCY_HEADER, got nil")
	}
	if !strings.Contains(err.Error(), "IDEMPOTENCY_HEADER") {
		t.Errorf("error = %q, want mention of IDEMPOTENCY_HEADER", err)
	}
}

func TestLoadMissingJWTSecret(t *testing.T) {
	old := setRequiredEnv()
	defer restoreEnv(old)
	_ = os.Setenv("JWT_SECRET", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for missing JWT_SECRET, got nil")
	}
	if !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Errorf("error = %q, want mention of JWT_SECRET", err)
	}
}

func TestLoadMissingServerAddress(t *testing.T) {
	old := setRequiredEnv()
	defer restoreEnv(old)
	_ = os.Setenv("SERVER_ADDRESS", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for missing SERVER_ADDRESS, got nil")
	}
	if !strings.Contains(err.Error(), "SERVER_ADDRESS") {
		t.Errorf("error = %q, want mention of SERVER_ADDRESS", err)
	}
}

func mustGetwd(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() failed: %v", err)
	}
	return wd
}
