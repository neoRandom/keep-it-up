package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	// LoadEnv reads ".env" if present; ensure a missing file doesn't break the test.
	oldJWT := os.Getenv("JWT_SECRET")
	oldAddr := os.Getenv("SERVER_ADDRESS")
	oldDB := os.Getenv("GOOSE_DBSTRING")
	defer func() {
		_ = os.Setenv("JWT_SECRET", oldJWT)
		_ = os.Setenv("SERVER_ADDRESS", oldAddr)
		_ = os.Setenv("GOOSE_DBSTRING", oldDB)
	}()

	_ = os.Setenv("JWT_SECRET", "secret")
	_ = os.Setenv("SERVER_ADDRESS", ":8080")
	_ = os.Setenv("GOOSE_DBSTRING", "db.sqlite")

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
}

func TestLoadMissingJWTSecret(t *testing.T) {
	oldJWT := os.Getenv("JWT_SECRET")
	_ = os.Setenv("JWT_SECRET", "")
	defer func() { _ = os.Setenv("JWT_SECRET", oldJWT) }()

	// Ensure the other required vars are set so only JWT_SECRET is missing.
	oldAddr := os.Getenv("SERVER_ADDRESS")
	oldDB := os.Getenv("GOOSE_DBSTRING")
	defer func() {
		_ = os.Setenv("SERVER_ADDRESS", oldAddr)
		_ = os.Setenv("GOOSE_DBSTRING", oldDB)
	}()
	_ = os.Setenv("SERVER_ADDRESS", ":8080")
	_ = os.Setenv("GOOSE_DBSTRING", "db.sqlite")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for missing JWT_SECRET, got nil")
	}
	if !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Errorf("error = %q, want mention of JWT_SECRET", err)
	}
}

func TestLoadMissingServerAddress(t *testing.T) {
	oldAddr := os.Getenv("SERVER_ADDRESS")
	_ = os.Setenv("SERVER_ADDRESS", "")
	defer func() { _ = os.Setenv("SERVER_ADDRESS", oldAddr) }()

	oldJWT := os.Getenv("JWT_SECRET")
	oldDB := os.Getenv("GOOSE_DBSTRING")
	defer func() {
		_ = os.Setenv("JWT_SECRET", oldJWT)
		_ = os.Setenv("GOOSE_DBSTRING", oldDB)
	}()
	_ = os.Setenv("JWT_SECRET", "secret")
	_ = os.Setenv("GOOSE_DBSTRING", "db.sqlite")

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