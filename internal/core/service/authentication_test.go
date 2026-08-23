package service_test

import (
	"keep-it-up/internal/core/service"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// TestAuthentication_IsPasswordValid exercises the password validity rule boundaries:
// minimum 6 characters and no whitespace.
func TestAuthentication_IsPasswordValid(t *testing.T) {
	tests := []struct {
		password string
		valid    bool
	}{
		// Length boundary.
		{"abc12", false},  // 5 chars - below minimum
		{"abc123", true},  // 6 chars - at minimum
		{"abc1234", true}, // 7 chars - above minimum
		{"", false},       // empty
		{"short", false},  // < 6 chars
		{"123456", true},  // 6 digits
		{"aaaaaa", true},  // 6 same letters
		// Whitespace.
		{" abc123", false},    // leading space
		{"abc123 ", false},    // trailing space
		{"secret 123", false}, // interior space
	}

	for _, tc := range tests {
		err := service.IsPasswordValid(tc.password)
		if tc.valid && err != nil {
			t.Fatalf("IsPasswordValid(%q) returned error: %v", tc.password, err)
		}
		if !tc.valid && err == nil {
			t.Fatalf("IsPasswordValid(%q) accepted an invalid password", tc.password)
		}
	}
}

// TestAuthentication_GeneratePasswordHash covers hashing behavior: valid passwords
// hash to a non-empty, non-raw, bcrypt-verifiable value; short passwords are
// rejected; and bcrypt salt makes repeated hashes of the same password differ
// while both still verify against it.
func TestAuthentication_GeneratePasswordHash(t *testing.T) {
	for _, password := range []string{"abc123", "secret123", "password", "aaaaaaaaa", "123456789"} {
		hash, err := service.GeneratePasswordHash(password)
		if err != nil {
			t.Fatalf("GeneratePasswordHash(%q) returned error: %v", password, err)
		}
		if hash == "" {
			t.Fatalf("GeneratePasswordHash(%q) returned empty hash", password)
		}
		if hash == password {
			t.Fatalf("GeneratePasswordHash(%q) returned unhashed password", password)
		}
		if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
			t.Fatalf("bcrypt compare failed for %q: %v", password, err)
		}
	}

	// A short password is rejected before hashing.
	if _, err := service.GeneratePasswordHash("short"); err == nil {
		t.Fatal("GeneratePasswordHash accepted a password shorter than 6 characters")
	}

	// bcrypt salt makes two hashes of the same password differ, but both verify.
	h1, err1 := service.GeneratePasswordHash("secret123")
	h2, err2 := service.GeneratePasswordHash("secret123")
	if err1 != nil || err2 != nil {
		t.Fatalf("GeneratePasswordHash returned error: err1=%v, err2=%v", err1, err2)
	}
	if h1 == h2 {
		t.Fatal("GeneratePasswordHash produced identical hashes for the same password (bcrypt should salt)")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(h1), []byte("secret123")); err != nil {
		t.Fatalf("first hash does not verify: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(h2), []byte("secret123")); err != nil {
		t.Fatalf("second hash does not verify: %v", err)
	}
}
