package usecase

import (
	"context"
	"strings"
	"testing"

	"keep-it-up/internal/core/model"
	"keep-it-up/internal/infrastructure/database"
)

func newTestPlayerManagement(t *testing.T, q *database.Queries) *PlayerManagement {
	t.Helper()
	uc, err := NewPlayerManagement(q, mustNewAuthentication(t, q, stubTG))
	if err != nil {
		t.Fatalf("NewPlayerManagement() returned error: %v", err)
	}
	return uc
}

func TestPlayerManagement_AddPlayer(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	uc := newTestPlayerManagement(t, queries)

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
	uc := newTestPlayerManagement(t, queries)

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
	uc := newTestPlayerManagement(t, queries)

	created, err := uc.AddPlayer(ctx, "Alice", "alice", "secret123")
	if err != nil {
		t.Fatalf("setup: AddPlayer() returned error: %v", err)
	}

	if err := uc.UpdatePlayerPassword(ctx, created.Username, "secret123", "newpass123"); err != nil {
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

func TestPlayerManagement_TrimPasswordWhitespaceBeforeStorage(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	uc := newTestPlayerManagement(t, queries)

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

		if player, err := mustNewAuthentication(t, queries, stubTG).CheckPlayerPassword(
			ctx, tc.username, strings.TrimSpace(tc.password),
		); err != nil || player.ID == 0 {
			t.Fatalf("CheckPlayerPassword() should accept username=%q with trimmed password=%q: ok=%v err=%v",
				tc.username, strings.TrimSpace(tc.password), player.ID == 0, err,
			)
		}
	}
}

func TestPlayerManagement_DuplicateUsername(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	uc := newTestPlayerManagement(t, queries)

	if _, err := uc.AddPlayer(ctx, "Alice", "alice", "secret123"); err != nil {
		t.Fatalf("AddPlayer() returned error for first player: %v", err)
	}
	if _, err := uc.AddPlayer(ctx, "Bob", "alice", "secret456"); err == nil {
		t.Fatal("AddPlayer() accepted duplicate username")
	}
}

func TestPlayerManagement_RejectsInvalidIDs(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	auth := mustNewAuthentication(t, queries, stubTG)
	uc, err := NewPlayerManagement(queries, auth)
	if err != nil {
		t.Fatalf("NewPlayerManagement() returned error: %v", err)
	}

	for _, id := range []int64{0, -1} {
		if err := uc.UpdatePlayerName(ctx, id, "Alice"); err == nil {
			t.Fatalf("UpdatePlayerName() accepted invalid id=%d", id)
		}
		if err := uc.BaseUpdatePlayerPassword(ctx, id, "secret123"); err == nil {
			t.Fatalf("BaseUpdatePlayerPassword() accepted invalid id=%d", id)
		}
		if err := uc.DeletePlayer(ctx, id); err == nil {
			t.Fatalf("DeletePlayer() accepted invalid id=%d", id)
		}
	}
}

func TestPlayerManagement_ForcePasswordUpdateBypassesPreviousPasswordCheck(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	uc := newTestPlayerManagement(t, queries)

	created, err := uc.AddPlayer(ctx, "Carol", "carol", "secret123")
	if err != nil {
		t.Fatalf("AddPlayer() returned error: %v", err)
	}

	if err := uc.UpdatePlayerPasswordForce(ctx, created.Username, "newpass456"); err != nil {
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
	uc := newTestPlayerManagement(t, queries)

	created, err := uc.AddPlayer(ctx, "Dana", "dana", "secret123")
	if err != nil {
		t.Fatalf("AddPlayer() returned error: %v", err)
	}

	if err := uc.UpdatePlayerPassword(ctx, created.Username, "wrongpassword", "newpass456"); err == nil {
		t.Fatal("UpdatePlayerPassword() accepted an incorrect previous password")
	}
}

func TestPlayerManagement_DeletePlayer(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	uc := newTestPlayerManagement(t, queries)

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

	uc, err := NewPlayerManagement(queries, nil)
	if err == nil {
		t.Fatal("NewPlayerManagement() should return an error when the auth dependency is missing")
	}
	if uc != nil {
		t.Fatal("NewPlayerManagement() should return a nil use case when construction fails")
	}
}

func TestPlayerManagement_RejectsInvalidInput(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	uc := newTestPlayerManagement(t, queries)

	// Name: minimum 2 chars; non-alphanumeric and whitespace are allowed.
	if _, err := uc.AddPlayer(ctx, "A", "alice", "secret123"); err == nil {
		t.Fatal("AddPlayer() accepted a short name")
	}
	if _, err := uc.AddPlayer(ctx, "Bad-Name", "bob", "secret123"); err != nil {
		t.Fatal("AddPlayer() rejected a non-alphanumeric name")
	}
	if _, err := uc.AddPlayer(ctx, "Alice Smith", "alice2", "secret123"); err != nil {
		t.Fatal("AddPlayer() rejected a name containing whitespace")
	}
	if err := uc.UpdatePlayerName(ctx, 1, "A"); err == nil {
		t.Fatal("UpdatePlayerName() accepted a short name")
	}

	// Username: minimum 3 chars and purely alphanumeric.
	for _, username := range []string{"ab", "alice smith", "bad-user", "user@123"} {
		if _, err := uc.AddPlayer(ctx, "ValidName", username, "secret123"); err == nil {
			t.Fatalf("AddPlayer() accepted invalid username: %q", username)
		}
	}

	// Password rule (min 6 chars, letters and digits only) lives in the service layer;
	// core/service/authentication_test.go is the authoritative boundary spec.
	// These representative cases confirm the use case propagates that rule.
	if _, err := uc.AddPlayer(ctx, "Player1", "player1", "abc12"); err == nil {
		t.Fatal("AddPlayer() accepted a password below the minimum length")
	}
	if _, err := uc.AddPlayer(ctx, "Player2", "player2", "secret 123"); err == nil {
		t.Fatal("AddPlayer() accepted a password containing a non-alphanumeric character")
	}
	if err := uc.UpdatePlayerPassword(ctx, "alice", "secret123", "short"); err == nil {
		t.Fatal("UpdatePlayerPassword() accepted a short password")
	}
}

func TestPlayerManagement_SuccessfulPlayerCRUDWithPasswordChange(t *testing.T) {
	ctx := context.Background()
	queries := newTestDB(t)
	uc := newTestPlayerManagement(t, queries)
	auth := mustNewAuthentication(t, queries, stubTG)

	player, err := uc.AddPlayer(ctx, "TestUser", "testuser", "password123")
	if err != nil {
		t.Fatalf("AddPlayer() returned error: %v", err)
	}
	if player.Name != "TestUser" {
		t.Fatalf("Player name = %q, want %q", player.Name, "TestUser")
	}

	player, err = auth.CheckPlayerPassword(ctx, "testuser", "password123")
	if err != nil || player.ID == 0 {
		t.Fatalf("CheckPlayerPassword() should succeed: ok=%v err=%v", player.ID == 0, err)
	}

	if err := uc.UpdatePlayerName(ctx, player.ID, "TestUserRenamed"); err != nil {
		t.Fatalf("UpdatePlayerName() returned error: %v", err)
	}

	if err := uc.UpdatePlayerPassword(ctx, player.Username, "password123", "newpassword456"); err != nil {
		t.Fatalf("UpdatePlayerPassword() returned error: %v", err)
	}

	player, err = auth.CheckPlayerPassword(ctx, "testuser", "password123")
	if err == nil && player.ID != 0 {
		t.Fatal("CheckPlayerPassword() should fail with old password")
	}

	player, err = auth.CheckPlayerPassword(ctx, "testuser", "newpassword456")
	if err != nil || player.ID == 0 {
		t.Fatalf(
			"CheckPlayerPassword() should succeed with new password: ok=%v err=%v",
			player.ID == 0, err,
		)
	}

	if err := uc.DeletePlayer(ctx, player.ID); err != nil {
		t.Fatalf("DeletePlayer() returned error: %v", err)
	}

	if _, err := queries.GetPlayer(ctx, player.ID); err == nil {
		t.Fatal("GetPlayer() found player after DeletePlayer()")
	}
}

func TestPlayerManagement_RejectsNilDependencies(t *testing.T) {
	queries := newTestDB(t)
	auth := mustNewAuthentication(t, queries, stubTG)

	if _, err := NewPlayerManagement(nil, auth); err == nil {
		t.Fatal("NewPlayerManagement() should return an error for nil queries")
	}
}

func TestPlayerManagement_CancelledContext(t *testing.T) {
	tests := []struct {
		name string
		run  func(ctx context.Context, uc *PlayerManagement, p model.Player) error
	}{
		{"AddPlayer", func(ctx context.Context, uc *PlayerManagement, _ model.Player) error {
			_, err := uc.AddPlayer(ctx, "Alice", "alice", "secret123")
			return err
		}},
		{"UpdatePlayerName", func(ctx context.Context, uc *PlayerManagement, p model.Player) error {
			return uc.UpdatePlayerName(ctx, p.ID, "AliceRenamed")
		}},
		{"UpdatePlayerPassword", func(ctx context.Context, uc *PlayerManagement, p model.Player) error {
			return uc.UpdatePlayerPassword(ctx, p.Username, "secret123", "newpass456")
		}},
		{"BaseUpdatePlayerPassword", func(ctx context.Context, uc *PlayerManagement, p model.Player) error {
			return uc.BaseUpdatePlayerPassword(ctx, p.ID, "newpass456")
		}},
		{"UpdatePlayerPasswordForce", func(ctx context.Context, uc *PlayerManagement, p model.Player) error {
			return uc.UpdatePlayerPasswordForce(ctx, p.Username, "newpass456")
		}},
		{"DeletePlayer", func(ctx context.Context, uc *PlayerManagement, p model.Player) error {
			return uc.DeletePlayer(ctx, p.ID)
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			queries := newTestDB(t)
			uc := newTestPlayerManagement(t, queries)
			p, err := uc.AddPlayer(context.Background(), "Seed", "seed", "secret123")
			if err != nil {
				t.Fatalf("setup AddPlayer: %v", err)
			}

			cctx, cancel := context.WithCancel(context.Background())
			cancel()

			if err := tc.run(cctx, uc, p); err == nil {
				t.Fatalf("%s() should return error for cancelled context", tc.name)
			}
		})
	}
}
