package usecase

import (
	"context"
	"testing"

	"keep-it-up/internal/infrastructure/database"

	"golang.org/x/crypto/bcrypt"
)

func TestAuthentication_VerifyPlayerPassword(t *testing.T) {
	auth := NewAuthentication(nil)

	for _, password := range []string{"secret123", "abc123", " secret123 ", "secret123 ", " secret123", "\tsecret123\n"} {
		if err := auth.VerifyPlayerPassword(password); err != nil {
			t.Fatalf("VerifyPlayerPassword() rejected a valid password %q: %v", password, err)
		}
	}

	for _, password := range []string{"", "short", "abc12", "pa ss", "p ass ", "secret 123", "pass  ", "alice", " alice ", "\tsecret 123\n"} {
		if err := auth.VerifyPlayerPassword(password); err == nil {
			t.Fatalf("VerifyPlayerPassword() accepted an invalid password %q", password)
		}
	}
}

func TestAuthentication_VerifyPlayerPasswordBoundary(t *testing.T) {
	// Test boundary condition: exactly 6 characters (minimum valid length)
	auth := NewAuthentication(nil)

	if err := auth.VerifyPlayerPassword("abc123"); err != nil {
		t.Fatalf("VerifyPlayerPassword() rejected valid 6-character password: %v", err)
	}

	// Test one below boundary: 5 characters (should fail)
	if err := auth.VerifyPlayerPassword("abc12"); err == nil {
		t.Fatal("VerifyPlayerPassword() accepted 5-character password (below minimum)")
	}

	// Test with whitespace at boundary
	if err := auth.VerifyPlayerPassword(" abc123"); err != nil {
		t.Fatalf("VerifyPlayerPassword() rejected valid 6-char password with leading space: %v", err)
	}
	if err := auth.VerifyPlayerPassword("abc123 "); err != nil {
		t.Fatalf("VerifyPlayerPassword() rejected valid 6-char password with trailing space: %v", err)
	}
}

func TestAuthentication_GeneratePasswordHash(t *testing.T) {
	auth := NewAuthentication(nil)

	hash, err := auth.GeneratePasswordHash("secret123")
	if err != nil {
		t.Fatalf("GeneratePasswordHash() returned error: %v", err)
	}
	if hash == "" {
		t.Fatal("GeneratePasswordHash() returned empty hash")
	}
	if hash == "secret123" {
		t.Fatal("GeneratePasswordHash() returned the raw password instead of a hash")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("secret123")); err != nil {
		t.Fatalf("bcrypt.CompareHashAndPassword() failed for generated hash: %v", err)
	}
}

func TestAuthentication_GeneratePasswordHashRejectsShortPassword(t *testing.T) {
	auth := NewAuthentication(nil)

	if _, err := auth.GeneratePasswordHash("short"); err == nil {
		t.Fatal("GeneratePasswordHash() accepted a password shorter than 6 characters")
	}
}

func TestAuthentication_CheckPlayerPasswordUsesPasswordRuleValidation(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	auth := NewAuthentication(queries)

	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword() returned error: %v", err)
	}

	_, err = queries.CreatePlayer(ctx, database.CreatePlayerParams{
		Name:           "Alice",
		Username:       "alice",
		HashedPassword: string(hash),
	})
	if err != nil {
		t.Fatalf("CreatePlayer() returned error: %v", err)
	}

	if _, err := auth.CheckPlayerPassword(ctx, "alice", "short"); err == nil {
		t.Fatal("CheckPlayerPassword() accepted a password that fails business-rule validation")
	}
	if _, err := auth.CheckPlayerPassword(ctx, "alice", "alice"); err == nil {
		t.Fatal("CheckPlayerPassword() accepted a password equal to the username: username='alice', supplied password='alice'")
	}
	if _, err := auth.CheckPlayerPassword(ctx, "alice", " alice "); err == nil {
		t.Fatal("CheckPlayerPassword() accepted a password that matches the username after trimming whitespace: username='alice', supplied password=' alice '")
	}
}

func TestAuthentication_CheckPlayerPassword(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	auth := NewAuthentication(queries)

	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword() returned error: %v", err)
	}

	_, err = queries.CreatePlayer(ctx, database.CreatePlayerParams{
		Name:           "Alice",
		Username:       "alice",
		HashedPassword: string(hash),
	})
	if err != nil {
		t.Fatalf("CreatePlayer() returned error: %v", err)
	}

	ok, err := auth.CheckPlayerPassword(ctx, "alice", "secret123")
	if err != nil {
		t.Fatalf("CheckPlayerPassword() returned error: %v", err)
	}
	if !ok {
		t.Fatal("CheckPlayerPassword() returned false for the correct password")
	}
}

func TestAuthentication_CheckPlayerPasswordRejectsWrongPassword(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	auth := NewAuthentication(queries)

	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword() returned error: %v", err)
	}

	_, err = queries.CreatePlayer(ctx, database.CreatePlayerParams{
		Name:           "Alice",
		Username:       "alice",
		HashedPassword: string(hash),
	})
	if err != nil {
		t.Fatalf("CreatePlayer() returned error: %v", err)
	}

	ok, err := auth.CheckPlayerPassword(ctx, "alice", "wrongpass")
	if err != nil {
		t.Fatalf("CheckPlayerPassword() returned error for an incorrect password: %v", err)
	}
	if ok {
		t.Fatal("CheckPlayerPassword() returned true for the wrong password")
	}
}

func TestAuthentication_CheckPlayerPasswordRejectsUnknownUser(t *testing.T) {
	ctx := context.Background()
	auth := NewAuthentication(newTestDB(t))

	ok, err := auth.CheckPlayerPassword(ctx, "ghost", "secret123")
	if err == nil {
		t.Fatal("CheckPlayerPassword() did not return an error for an unknown username")
	}
	if ok {
		t.Fatal("CheckPlayerPassword() returned true for an unknown username")
	}
}

func TestAuthentication_CheckPlayerPasswordRejectsInvalidInput(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	auth := NewAuthentication(queries)

	if _, err := auth.CheckPlayerPassword(ctx, "a", "secret123"); err == nil {
		t.Fatal("CheckPlayerPassword() accepted a short username")
	}

	if _, err := auth.CheckPlayerPassword(ctx, "alice", "short"); err == nil {
		t.Fatal("CheckPlayerPassword() accepted a short password")
	}

	if _, err := auth.CheckPlayerPassword(ctx, "alice", "alice"); err == nil {
		t.Fatal("CheckPlayerPassword() accepted a password equal to the username")
	}

	authNil := NewAuthentication(nil)
	if _, err := authNil.CheckPlayerPassword(ctx, "alice", "secret123"); err == nil {
		t.Fatal("CheckPlayerPassword() accepted nil database queries")
	}
}

func TestAuthentication_CheckPlayerPasswordRejectsMalformedHash(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	auth := NewAuthentication(queries)

	_, err := queries.CreatePlayer(ctx, database.CreatePlayerParams{
		Name:           "Alice",
		Username:       "alice",
		HashedPassword: "not-a-valid-bcrypt-hash",
	})
	if err != nil {
		t.Fatalf("CreatePlayer() returned error: %v", err)
	}

	ok, err := auth.CheckPlayerPassword(ctx, "alice", "secret123")
	if err == nil {
		t.Fatal("CheckPlayerPassword() did not return an error for a malformed stored hash")
	}
	if ok {
		t.Fatal("CheckPlayerPassword() returned true for a malformed stored hash")
	}
}

func TestAuthentication_CheckPlayerPasswordWithCancelledContext(t *testing.T) {
	queries := newTestDB(t)
	auth := NewAuthentication(queries)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := auth.CheckPlayerPassword(ctx, "alice", "secret123")
	if err == nil {
		t.Fatal("CheckPlayerPassword() should return error for cancelled context")
	}
}

func TestAuthentication_VerifyPlayerPasswordBoundaryPrecision(t *testing.T) {
	auth := NewAuthentication(nil)

	// Test exactly at boundary: 6 characters (minimum valid)
	// Note: This test documents desired behavior for password validation boundaries
	testCases := []struct {
		password       string
		shouldValidate bool
	}{
		{"abc123", true},  // exactly 6 chars - at boundary
		{"abc12", false},  // 5 chars - one below minimum
		{"abc1234", true}, // 7 chars - one above minimum
		{"123456", true},  // 6 digits
		{"aaaaaa", true},  // 6 of same letter
	}

	for _, tc := range testCases {
		err := auth.VerifyPlayerPassword(tc.password)
		if tc.shouldValidate && err != nil {
			t.Fatalf("VerifyPlayerPassword() should validate %q (length=%d): %v", tc.password, len(tc.password), err)
		}
		if !tc.shouldValidate && err == nil {
			t.Fatalf("VerifyPlayerPassword() should reject %q (length=%d)", tc.password, len(tc.password))
		}
	}
}

func TestAuthentication_GeneratePasswordHashWithValidPasswords(t *testing.T) {
	auth := NewAuthentication(nil)

	validPasswords := []string{
		"abc123",    // exactly 6 chars
		"secret123", // 9 chars
		"password",  // 8 chars
		"aaaaaaaaa", // 9 of same character
		"123456789", // 9 digits
	}

	for _, password := range validPasswords {
		hash, err := auth.GeneratePasswordHash(password)
		if err != nil {
			t.Fatalf("GeneratePasswordHash() failed for valid password %q: %v", password, err)
		}
		if hash == "" {
			t.Fatalf("GeneratePasswordHash() returned empty hash for %q", password)
		}
		if hash == password {
			t.Fatalf("GeneratePasswordHash() returned unhashed password for %q", password)
		}
	}
}

func TestAuthentication_CheckPlayerPasswordWithTrimmablePassword(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	auth := NewAuthentication(queries)

	hash, _ := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	_, err := queries.CreatePlayer(ctx, database.CreatePlayerParams{
		Name:           "Alice",
		Username:       "alice",
		HashedPassword: string(hash),
	})
	if err != nil {
		t.Fatalf("CreatePlayer() failed: %v", err)
	}

	// Test that exact password match works
	ok, err := auth.CheckPlayerPassword(ctx, "alice", "secret123")
	if err != nil {
		t.Fatalf("CheckPlayerPassword() failed: %v", err)
	}
	if !ok {
		t.Fatal("CheckPlayerPassword() should accept exact password match")
	}

	// Test that passwords with leading/trailing spaces are handled by trimming in VerifyPlayerPassword
	// Note: This test documents how passwords with whitespace are handled
	for _, password := range []string{" secret123", "secret123 ", " secret123 ", "\tsecret123", "secret123\n"} {
		ok, err := auth.CheckPlayerPassword(ctx, "alice", password)
		// These should work IF VerifyPlayerPassword trims them before checking
		// The behavior depends on the implementation
		_ = ok
		_ = err
	}
}

func TestAuthentication_VerifyPlayerPasswordWithUnicodeCharacters(t *testing.T) {
	auth := NewAuthentication(nil)

	// Test that unicode characters are allowed (if validation allows non-ASCII)
	// The exact behavior depends on the validation implementation
	testCases := []struct {
		password string
		desc     string
	}{
		{"пароль123", "Cyrillic password"},
		{"password123", "ASCII password"},
		{"пароль", "Cyrillic word only"},
	}

	for _, tc := range testCases {
		// Just verify the function doesn't panic or error unexpectedly
		_ = auth.VerifyPlayerPassword(tc.password)
	}
}

func TestAuthentication_GeneratePasswordHashIsConsistent(t *testing.T) {
	auth := NewAuthentication(nil)

	password := "secret123"
	hash1, err1 := auth.GeneratePasswordHash(password)
	hash2, err2 := auth.GeneratePasswordHash(password)

	if err1 != nil || err2 != nil {
		t.Fatalf("GeneratePasswordHash() returned error: err1=%v, err2=%v", err1, err2)
	}

	// Bcrypt hashes should be different even for same password (due to salt)
	if hash1 == hash2 {
		t.Fatal("GeneratePasswordHash() produced identical hashes for same password (should be different due to salt)")
	}

	// But both should validate against the same password
	if err := bcrypt.CompareHashAndPassword([]byte(hash1), []byte(password)); err != nil {
		t.Fatalf("First hash does not match password: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash2), []byte(password)); err != nil {
		t.Fatalf("Second hash does not match password: %v", err)
	}
}
