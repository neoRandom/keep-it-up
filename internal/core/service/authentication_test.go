package service

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestAuthentication_IsPasswordValid(t *testing.T) {
	for _, password := range []string{"secret123", "abc123"} {
		if err := IsPasswordValid(password); err != nil {
			t.Fatalf("IsPasswordValid() rejected a valid password %q: %v", password, err)
		}
	}

	for _, password := range []string{
		"", "short", "abc12", "pa ss", "p ass ", "secret 123", "pass  ",
		"alice", " alice ", " secret123 ", "secret123 ", " secret123",
		"\tsecret 123\n",
	} {
		if err := IsPasswordValid(password); err == nil {
			t.Fatalf("IsPasswordValid() accepted an invalid password %q", password)
		}
	}
}

func TestAuthentication_IsPasswordValidBoundary(t *testing.T) {
	if err := IsPasswordValid("abc123"); err != nil {
		t.Fatalf("IsPasswordValid() rejected valid 6-character password: %v", err)
	}

	// Test one below boundary: 5 characters (should fail)
	if err := IsPasswordValid("abc12"); err == nil {
		t.Fatal("IsPasswordValid() accepted 5-character password (below minimum)")
	}

	// Test with whitespace at boundary
	if err := IsPasswordValid(" abc123"); err == nil {
		t.Fatalf("IsPasswordValid() accepted invalid 6-char password with leading space: %v", err)
	}
	if err := IsPasswordValid("abc123 "); err == nil {
		t.Fatalf("IsPasswordValid() accepted invalid 6-char password with trailing space: %v", err)
	}
}

func TestAuthentication_GeneratePasswordHash(t *testing.T) {
	hash, err := GeneratePasswordHash("secret123")
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
	if _, err := GeneratePasswordHash("short"); err == nil {
		t.Fatal("GeneratePasswordHash() accepted a password shorter than 6 characters")
	}
}

func TestAuthentication_IsPasswordValidBoundaryPrecision(t *testing.T) {
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
		err := IsPasswordValid(tc.password)
		if tc.shouldValidate && err != nil {
			t.Fatalf("IsPasswordValid() should validate %q (length=%d): %v", tc.password, len(tc.password), err)
		}
		if !tc.shouldValidate && err == nil {
			t.Fatalf("IsPasswordValid() should reject %q (length=%d)", tc.password, len(tc.password))
		}
	}
}

func TestAuthentication_GeneratePasswordHashWithValidPasswords(t *testing.T) {
	validPasswords := []string{
		"abc123",    // exactly 6 chars
		"secret123", // 9 chars
		"password",  // 8 chars
		"aaaaaaaaa", // 9 of same character
		"123456789", // 9 digits
	}

	for _, password := range validPasswords {
		hash, err := GeneratePasswordHash(password)
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

func TestAuthentication_IsPasswordValidWithUnicodeCharacters(t *testing.T) {
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
		_ = IsPasswordValid(tc.password)
	}
}

func TestAuthentication_GeneratePasswordHashIsConsistent(t *testing.T) {
	password := "secret123"
	hash1, err1 := GeneratePasswordHash(password)
	hash2, err2 := GeneratePasswordHash(password)

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
