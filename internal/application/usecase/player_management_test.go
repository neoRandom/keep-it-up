package usecase

import (
	"context"
	"strings"
	"testing"
)

func TestPlayerManagement_AddPlayer(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	uc := NewPlayerManagement(queries, NewAuthentication(queries, nil))

	created, err := uc.AddPlayer(ctx, "Alice", "alice", "secret123")
	if err != nil {
		t.Fatalf("AddPlayer() returned error: %v", err)
	}
	if created.Name != "Alice" {
		t.Fatalf("AddPlayer() created name = %q, want %q", created.Name, "Alice")
	}
}

func TestPlayerManagement_UpdatePlayerName(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	uc := NewPlayerManagement(queries, NewAuthentication(queries, nil))

	created, err := uc.AddPlayer(ctx, "Alice", "alice", "secret123")
	if err != nil {
		t.Fatalf("setup: AddPlayer() returned error: %v", err)
	}

	if err := uc.UpdatePlayerName(ctx, created.ID, "Alicia"); err != nil {
		t.Fatalf("UpdatePlayerName() returned error: %v", err)
	}

	updated, err := queries.GetPlayer(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetPlayer() returned error: %v", err)
	}
	if updated.Name != "Alicia" {
		t.Fatalf("GetPlayer() name = %q, want %q", updated.Name, "Alicia")
	}
}

func TestPlayerManagement_UpdatePlayerPassword(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	uc := NewPlayerManagement(queries, NewAuthentication(queries, nil))

	created, err := uc.AddPlayer(ctx, "Alice", "alice", "secret123")
	if err != nil {
		t.Fatalf("setup: AddPlayer() returned error: %v", err)
	}

	if err := uc.UpdatePlayerPassword(ctx, created.ID, "secret123", "newpass123"); err != nil {
		t.Fatalf("UpdatePlayerPassword() returned error: %v", err)
	}

	updated, err := queries.GetPlayer(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetPlayer() returned error: %v", err)
	}
	if updated.HashedPassword == "newpass123" {
		t.Fatal("GetPlayer() stored a plain-text password instead of a hash")
	}
}

func TestPlayerManagement_RejectsInvalidPlayerInput(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	uc := NewPlayerManagement(queries, NewAuthentication(queries, nil))

	if _, err := uc.AddPlayer(ctx, "A", "alice", "secret123"); err == nil {
		t.Fatal("AddPlayer() accepted short name")
	}

	if _, err := uc.AddPlayer(ctx, "Alice", "al", "secret123"); err == nil {
		t.Fatal("AddPlayer() accepted short username")
	}

	if _, err := uc.AddPlayer(ctx, "Alice", "alice", "short"); err == nil {
		t.Fatal("AddPlayer() accepted short password")
	}

	if _, err := uc.AddPlayer(ctx, "Bad-Name", "bob", "secret123"); err != nil {
		t.Fatal("AddPlayer() rejected non-alphanumeric name")
	}

	if err := uc.UpdatePlayerName(ctx, 1, "A"); err == nil {
		t.Fatal("UpdatePlayerName() accepted short name")
	}

	if err := uc.UpdatePlayerPassword(ctx, 1, "secret123", "short"); err == nil {
		t.Fatal("UpdatePlayerPassword() accepted short password")
	}
}

func TestPlayerManagement_RejectsWhitespaceInNamesAndUsernames(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	uc := NewPlayerManagement(queries, NewAuthentication(queries, nil))

	if _, err := uc.AddPlayer(ctx, "Alice Smith", "alice", "secret123"); err != nil {
		t.Fatal("AddPlayer() rejected a name containing whitespace")
	}
	if _, err := uc.AddPlayer(ctx, "Alice", "alice smith", "secret123"); err == nil {
		t.Fatal("AddPlayer() accepted a username containing whitespace")
	}
}

func TestPlayerManagement_TrimPasswordWhitespaceBeforeStorage(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	uc := NewPlayerManagement(queries, NewAuthentication(queries, nil))

	for _, tc := range []struct {
		name     string
		username string
		password string
	}{
		{name: "Alice", username: "alice", password: " secret123 "},
		{name: "Bob", username: "bob", password: "secret123 "},
		{name: "Carol", username: "carol", password: " secret123"},
		{name: "Dana", username: "dana", password: "\tsecret123\n"},
	} {
		if _, err := uc.AddPlayer(ctx, tc.name, tc.username, tc.password); err != nil {
			t.Fatalf("AddPlayer() rejected a password that should be valid after trimming outer whitespace: %q (%v)", tc.password, err)
		}

		player, err := queries.GetPlayerByUsername(ctx, tc.username)
		if err != nil {
			t.Fatalf("GetPlayerByUsername() failed after AddPlayer() for %q: %v", tc.username, err)
		}
		if player.HashedPassword == tc.password {
			t.Fatalf("AddPlayer() stored the raw password with whitespace for %q", tc.password)
		}

		if ok, err := NewAuthentication(queries, nil).CheckPlayerPassword(ctx, tc.username, strings.TrimSpace(tc.password)); err != nil || !ok {
			t.Fatalf("CheckPlayerPassword() should accept username=%q with trimmed password=%q: ok=%v err=%v", tc.username, strings.TrimSpace(tc.password), ok, err)
		}
	}

	if _, err := uc.AddPlayer(ctx, "Eve", "eve", "alice"); err == nil {
		t.Fatal("AddPlayer() accepted a password that matches the username: username='eve', supplied password='alice'")
	}

	if _, err := uc.AddPlayer(ctx, "Eve", "eve", "eve"); err == nil {
		t.Fatal("AddPlayer() accepted a password that matches the username: username='eve', supplied password='eve'")
	}

	if _, err := uc.AddPlayer(ctx, "Eve", "eve", "alice"); err == nil {
		t.Fatal("AddPlayer() accepted a password equal to the username")
	}
	if _, err := uc.AddPlayer(ctx, "Eve", "eve", " alice "); err == nil {
		t.Fatal("AddPlayer() accepted a trimmed username-like password")
	}

	for _, password := range []string{"p ass", "secret 123", "pass  "} {
		if _, err := uc.AddPlayer(ctx, "Eve", "eve", password); err == nil {
			t.Fatalf("AddPlayer() accepted an invalid password with whitespace: %q", password)
		}
	}
}

func TestPlayerManagement_RejectsNilDependencies(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	auth := NewAuthentication(queries, nil)

	if _, err := (&PlayerManagement{q: queries}).AddPlayer(ctx, "Alice", "alice", "secret123"); err == nil {
		t.Fatal("AddPlayer() accepted nil auth dependency")
	}
	if err := (&PlayerManagement{auth: auth}).UpdatePlayerName(ctx, 1, "Alice"); err == nil {
		t.Fatal("UpdatePlayerName() accepted nil query dependency")
	}
	if err := (&PlayerManagement{auth: auth}).BaseUpdatePlayerPassword(ctx, 1, "secret123"); err == nil {
		t.Fatal("BaseUpdatePlayerPassword() accepted nil query dependency")
	}
	if err := (&PlayerManagement{auth: auth}).DeletePlayer(ctx, 1); err == nil {
		t.Fatal("DeletePlayer() accepted nil query dependency")
	}
}

func TestPlayerManagement_RejectsInvalidIDs(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	auth := NewAuthentication(queries, nil)
	uc := NewPlayerManagement(queries, auth)

	for _, id := range []int64{0, -1} {
		if err := uc.UpdatePlayerName(ctx, id, "Alice"); err == nil {
			t.Fatalf("UpdatePlayerName() accepted invalid id=%d", id)
		}
		if err := uc.BaseUpdatePlayerPassword(ctx, id, "secret123"); err == nil {
			t.Fatalf("BaseUpdatePlayerPassword() accepted invalid id=%d", id)
		}
		if err := uc.UpdatePlayerPassword(ctx, id, "secret123", "newpass123"); err == nil {
			t.Fatalf("UpdatePlayerPassword() accepted invalid id=%d", id)
		}
		if err := uc.UpdatePlayerPasswordForce(ctx, id, "newpass123"); err == nil {
			t.Fatalf("UpdatePlayerPasswordForce() accepted invalid id=%d", id)
		}
		if err := uc.DeletePlayer(ctx, id); err == nil {
			t.Fatalf("DeletePlayer() accepted invalid id=%d", id)
		}
	}
}

func TestPlayerManagement_ForcePasswordUpdateBypassesPreviousPasswordCheck(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	uc := NewPlayerManagement(queries, NewAuthentication(queries, nil))

	created, err := uc.AddPlayer(ctx, "Carol", "carol", "secret123")
	if err != nil {
		t.Fatalf("AddPlayer() returned error: %v", err)
	}

	if err := uc.UpdatePlayerPasswordForce(ctx, created.ID, "newpass456"); err != nil {
		t.Fatalf("UpdatePlayerPasswordForce() returned error: %v", err)
	}

	player, err := queries.GetPlayer(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetPlayer() returned error: %v", err)
	}
	if player.HashedPassword == "newpass456" {
		t.Fatal("UpdatePlayerPasswordForce() stored a plain-text password")
	}
}

func TestPlayerManagement_RejectsWrongPreviousPassword(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	uc := NewPlayerManagement(queries, NewAuthentication(queries, nil))

	created, err := uc.AddPlayer(ctx, "Dana", "dana", "secret123")
	if err != nil {
		t.Fatalf("AddPlayer() returned error: %v", err)
	}

	if err := uc.UpdatePlayerPassword(ctx, created.ID, "wrongpassword", "newpass456"); err == nil {
		t.Fatal("UpdatePlayerPassword() accepted an incorrect previous password")
	}
}

func TestPlayerManagement_DeletePlayer(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	uc := NewPlayerManagement(queries, NewAuthentication(queries, nil))

	created, err := uc.AddPlayer(ctx, "Bob", "bob", "secret123")
	if err != nil {
		t.Fatalf("AddPlayer() returned error: %v", err)
	}

	if err := uc.DeletePlayer(ctx, created.ID); err != nil {
		t.Fatalf("DeletePlayer() returned error: %v", err)
	}

	if _, err := queries.GetPlayer(ctx, created.ID); err == nil {
		t.Fatal("GetPlayer() found deleted player")
	}
}

func TestNewPlayerManagement_RejectsNilAuthDependency(t *testing.T) {
	queries := newTestDB(t)

	if uc := NewPlayerManagement(queries, nil); uc != nil {
		t.Fatal("NewPlayerManagement() should return nil when the auth dependency is missing")
	}
}

func TestPlayerManagement_AddPlayerWithCancelledContext(t *testing.T) {
	queries := newTestDB(t)
	uc := NewPlayerManagement(queries, NewAuthentication(queries, nil))

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := uc.AddPlayer(ctx, "Alice", "alice", "secret123")
	if err == nil {
		t.Fatal("AddPlayer() should return error for cancelled context")
	}
}

func TestPlayerManagement_DuplicateUsername(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	uc := NewPlayerManagement(queries, NewAuthentication(queries, nil))

	// Add first player
	if _, err := uc.AddPlayer(ctx, "Alice", "alice", "secret123"); err != nil {
		t.Fatalf("AddPlayer() returned error for first player: %v", err)
	}

	// Try to add another player with same username
	if _, err := uc.AddPlayer(ctx, "Bob", "alice", "secret456"); err == nil {
		t.Fatal("AddPlayer() accepted duplicate username")
	}
}

func TestPlayerManagement_SuccessfulPlayerCRUDWithPasswordChange(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	uc := NewPlayerManagement(queries, NewAuthentication(queries, nil))
	auth := NewAuthentication(queries, nil)

	// Create
	player, err := uc.AddPlayer(ctx, "TestUser", "testuser", "password123")
	if err != nil {
		t.Fatalf("AddPlayer() returned error: %v", err)
	}
	if player.Name != "TestUser" {
		t.Fatalf("Player name = %q, want %q", player.Name, "TestUser")
	}

	// Verify can authenticate
	ok, err := auth.CheckPlayerPassword(ctx, "testuser", "password123")
	if err != nil || !ok {
		t.Fatalf("CheckPlayerPassword() should succeed: ok=%v err=%v", ok, err)
	}

	// Update name
	if err := uc.UpdatePlayerName(ctx, player.ID, "TestUserRenamed"); err != nil {
		t.Fatalf("UpdatePlayerName() returned error: %v", err)
	}

	// Update password using old password
	if err := uc.UpdatePlayerPassword(ctx, player.ID, "password123", "newpassword456"); err != nil {
		t.Fatalf("UpdatePlayerPassword() returned error: %v", err)
	}

	// Verify old password no longer works
	ok, err = auth.CheckPlayerPassword(ctx, "testuser", "password123")
	if err == nil && ok {
		t.Fatal("CheckPlayerPassword() should fail with old password")
	}

	// Verify new password works
	ok, err = auth.CheckPlayerPassword(ctx, "testuser", "newpassword456")
	if err != nil || !ok {
		t.Fatalf("CheckPlayerPassword() should succeed with new password: ok=%v err=%v", ok, err)
	}

	// Delete
	if err := uc.DeletePlayer(ctx, player.ID); err != nil {
		t.Fatalf("DeletePlayer() returned error: %v", err)
	}

	// Verify deletion
	if _, err := queries.GetPlayer(ctx, player.ID); err == nil {
		t.Fatal("GetPlayer() found player after DeletePlayer()")
	}
}

func TestPlayerManagement_UpdatePlayerNameWithCancelledContext(t *testing.T) {
	queries := newTestDB(t)
	uc := NewPlayerManagement(queries, NewAuthentication(queries, nil))

	created, _ := uc.AddPlayer(context.Background(), "Alice", "alice", "secret123")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := uc.UpdatePlayerName(ctx, created.ID, "AliceRenamed"); err == nil {
		t.Fatal("UpdatePlayerName() should return error for cancelled context")
	}
}

func TestPlayerManagement_UpdatePlayerPasswordWithCancelledContext(t *testing.T) {
	queries := newTestDB(t)
	uc := NewPlayerManagement(queries, NewAuthentication(queries, nil))

	created, _ := uc.AddPlayer(context.Background(), "Bob", "bob", "secret123")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := uc.UpdatePlayerPassword(ctx, created.ID, "secret123", "newpass456"); err == nil {
		t.Fatal("UpdatePlayerPassword() should return error for cancelled context")
	}
}

func TestPlayerManagement_BaseUpdatePlayerPasswordWithCancelledContext(t *testing.T) {
	queries := newTestDB(t)
	uc := NewPlayerManagement(queries, NewAuthentication(queries, nil))

	created, _ := uc.AddPlayer(context.Background(), "Carol", "carol", "secret123")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := uc.BaseUpdatePlayerPassword(ctx, created.ID, "newpass456"); err == nil {
		t.Fatal("BaseUpdatePlayerPassword() should return error for cancelled context")
	}
}

func TestPlayerManagement_UpdatePlayerPasswordForceWithCancelledContext(t *testing.T) {
	queries := newTestDB(t)
	uc := NewPlayerManagement(queries, NewAuthentication(queries, nil))

	created, _ := uc.AddPlayer(context.Background(), "Dana", "dana", "secret123")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := uc.UpdatePlayerPasswordForce(ctx, created.ID, "newpass456"); err == nil {
		t.Fatal("UpdatePlayerPasswordForce() should return error for cancelled context")
	}
}

func TestPlayerManagement_DeletePlayerWithCancelledContext(t *testing.T) {
	queries := newTestDB(t)
	uc := NewPlayerManagement(queries, NewAuthentication(queries, nil))

	created, _ := uc.AddPlayer(context.Background(), "Eve", "eve", "secret123")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := uc.DeletePlayer(ctx, created.ID); err == nil {
		t.Fatal("DeletePlayer() should return error for cancelled context")
	}
}

func TestPlayerManagement_RejectsInvalidUsernameInput(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	uc := NewPlayerManagement(queries, NewAuthentication(queries, nil))

	// Test AddPlayer with invalid usernames
	invalidUsernames := []string{
		"ab",       // too short
		"bad user", // contains space
		"bad-user", // contains hyphen
		"user@123", // contains special char
	}

	for _, username := range invalidUsernames {
		if _, err := uc.AddPlayer(ctx, "ValidName", username, "secret123"); err == nil {
			t.Fatalf("AddPlayer() accepted invalid username: %q", username)
		}
	}
}

func TestPlayerManagement_PasswordBoundaryValidation(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	uc := NewPlayerManagement(queries, NewAuthentication(queries, nil))

	// Test AddPlayer password validation at boundaries
	testCases := []struct {
		password string
		valid    bool
	}{
		{"abc123", true},      // exactly 6 chars - boundary
		{"abc12", false},      // 5 chars - below boundary
		{"secret123", true},   // 9 chars - above boundary
		{"secret 123", false}, // has space in middle
	}

	for idx, tc := range testCases {
		username := "user" + string('A'+rune(idx))
		_, err := uc.AddPlayer(ctx, "Player"+string('A'+rune(idx)), username, tc.password)
		if tc.valid && err != nil {
			t.Fatalf("AddPlayer() should accept password %q: %v", tc.password, err)
		}
		if !tc.valid && err == nil {
			t.Fatalf("AddPlayer() should reject password %q", tc.password)
		}
	}
}

func TestPlayerManagement_RejectsNilAuthDependencyInAllMethods(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)

	// Create a PlayerManagement with queries but without auth
	uc := &PlayerManagement{q: queries, auth: nil}

	// All methods should reject nil auth
	if _, err := uc.AddPlayer(ctx, "Alice", "alice", "secret123"); err == nil {
		t.Fatal("AddPlayer() accepted nil auth")
	}

	player, _ := NewPlayerManagement(queries, NewAuthentication(queries, nil)).AddPlayer(ctx, "Bob", "bob", "secret123")

	if err := uc.BaseUpdatePlayerPassword(ctx, player.ID, "newpass123"); err == nil {
		t.Fatal("BaseUpdatePlayerPassword() accepted nil auth")
	}

	if err := uc.UpdatePlayerPassword(ctx, player.ID, "secret123", "newpass123"); err == nil {
		t.Fatal("UpdatePlayerPassword() accepted nil auth")
	}

	if err := uc.UpdatePlayerPasswordForce(ctx, player.ID, "newpass123"); err == nil {
		t.Fatal("UpdatePlayerPasswordForce() accepted nil auth")
	}
}

func TestPlayerManagement_RejectsNilQueryDependencyInAllMethods(t *testing.T) {
	ctx := context.Background()
	auth := NewAuthentication(newTestDB(t), nil)

	// Create a PlayerManagement with auth but without queries
	uc := &PlayerManagement{q: nil, auth: auth}

	// All methods should reject nil queries
	if _, err := uc.AddPlayer(ctx, "Alice", "alice", "secret123"); err == nil {
		t.Fatal("AddPlayer() accepted nil queries")
	}

	if err := uc.UpdatePlayerName(ctx, 1, "Alice"); err == nil {
		t.Fatal("UpdatePlayerName() accepted nil queries")
	}

	if err := uc.BaseUpdatePlayerPassword(ctx, 1, "newpass123"); err == nil {
		t.Fatal("BaseUpdatePlayerPassword() accepted nil queries")
	}

	if err := uc.UpdatePlayerPassword(ctx, 1, "secret123", "newpass123"); err == nil {
		t.Fatal("UpdatePlayerPassword() accepted nil queries")
	}

	if err := uc.UpdatePlayerPasswordForce(ctx, 1, "newpass123"); err == nil {
		t.Fatal("UpdatePlayerPasswordForce() accepted nil queries")
	}

	if err := uc.DeletePlayer(ctx, 1); err == nil {
		t.Fatal("DeletePlayer() accepted nil queries")
	}
}
