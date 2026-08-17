package driver_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"keep-it-up/internal/infrastructure/database"
	"keep-it-up/internal/infrastructure/driver"
)

// --- Mocks ---

type mockGames struct {
	AddGameFunc    func(ctx context.Context, name string) (database.Game, error)
	UpdateGameFunc func(ctx context.Context, id int64, name string) error
	DeleteGameFunc func(ctx context.Context, id int64) error
}

func (m *mockGames) AddGame(ctx context.Context, name string) (database.Game, error) {
	return m.AddGameFunc(ctx, name)
}
func (m *mockGames) UpdateGame(ctx context.Context, id int64, name string) error {
	return m.UpdateGameFunc(ctx, id, name)
}
func (m *mockGames) DeleteGame(ctx context.Context, id int64) error {
	return m.DeleteGameFunc(ctx, id)
}

type mockCommands struct {
	SaveGameFunc   func(ctx context.Context, gameID, playerID int64, amount time.Time) error
	ResumeGameFunc func(ctx context.Context, gameID, playerID int64) error
	PauseGameFunc  func(ctx context.Context, gameID, playerID int64) error
}

func (m *mockCommands) SaveGame(ctx context.Context, gameID, playerID int64, amount time.Time) error {
	return m.SaveGameFunc(ctx, gameID, playerID, amount)
}
func (m *mockCommands) ResumeGame(ctx context.Context, gameID, playerID int64) error {
	return m.ResumeGameFunc(ctx, gameID, playerID)
}
func (m *mockCommands) PauseGame(ctx context.Context, gameID, playerID int64) error {
	return m.PauseGameFunc(ctx, gameID, playerID)
}

// --- Tests ---

func TestCLI_Run(t *testing.T) {
	ctx := context.Background()
	dummyErr := errors.New("business logic error")

	tests := []struct {
		name           string
		args           []string
		setupMocks     func(d *driver.Deps)
		expectedErr    error
		expectedStdout string
		expectedStderr string
	}{
		// Basic CLI checks
		{
			name:        "no arguments returns ErrNoCommand",
			args:        []string{},
			expectedErr: driver.ErrNoCommand,
			expectedStderr: "Usage:", 
		},
		{
			name:        "help flag returns usage on stdout",
			args:        []string{"--help"},
			expectedErr: nil,
			expectedStdout: "Usage:",
		},
		{
			name:        "unknown noun returns ErrUnknownCommand",
			args:        []string{"unknown-noun"},
			expectedErr: driver.ErrUnknownCommand,
			expectedStderr: "Usage:",
		},

		// Game subcommand tests
		{
			name: "game without verb returns ErrNoSubcommand",
			args: []string{"game"},
			expectedErr: driver.ErrNoSubcommand,
		},
		{
			name: "game add success",
			args: []string{"game", "add", "MyGame"},
			setupMocks: func(d *driver.Deps) {
				d.Games = &mockGames{
					AddGameFunc: func(ctx context.Context, name string) (database.Game, error) {
						if name != "MyGame" {
							t.Errorf("expected MyGame, got %s", name)
						}
						return database.Game{ID: 42, Name: "MyGame"}, nil 
					},
				}
			},
			expectedErr:    nil,
			expectedStdout: "{ID:42 Name:MyGame", 
		},
		{
			name: "game add fails wrong arg count",
			args: []string{"game", "add", "Too", "Many"},
			expectedErr: driver.ErrWrongArgCount,
		},
		{
			name: "game update invalid id",
			args: []string{"game", "update", "not-an-id", "NewName"},
			// Will be wrapped in parseID error, so we just check if it returns an error
			expectedErr: errors.New("invalid id"), 
		},

		// Session subcommand tests
		{
			name: "session save success",
			args: []string{"session", "save", "10", "20", "2026-08-17T09:46:15Z"},
			setupMocks: func(d *driver.Deps) {
				d.Commands = &mockCommands{
					SaveGameFunc: func(ctx context.Context, gameID, playerID int64, amount time.Time) error {
						if gameID != 10 || playerID != 20 {
							t.Errorf("unexpected IDs")
						}
						return nil
					},
				}
			},
			expectedErr:    nil,
			expectedStdout: "game 10 saved for player 20 at 2026-08-17T09:46:15Z\n",
		},
		{
			name: "session save with invalid timestamp",
			args: []string{"session", "save", "10", "20", "bad-time"},
			expectedErr: errors.New("invalid timestamp"),
		},
		{
			name: "session resume business error",
			args: []string{"session", "resume", "1", "2"},
			setupMocks: func(d *driver.Deps) {
				d.Commands = &mockCommands{
					ResumeGameFunc: func(ctx context.Context, gameID, playerID int64) error {
						return dummyErr
					},
				}
			},
			expectedErr: dummyErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			// Construct dependencies
			deps := driver.Deps{
				Stdout: &stdout,
				Stderr: &stderr,
			}

			// Apply test-specific mock setup
			if tt.setupMocks != nil {
				tt.setupMocks(&deps)
			}

			cliApp := driver.New(deps)
			err := cliApp.Run(ctx, tt.args)

			// Assert Errors
			if tt.expectedErr != nil {
				if err == nil {
					t.Fatalf("expected error containing %v, got nil", tt.expectedErr)
				}
				// Check for Sentinel Errors or substring match for wrapped errors
				if !errors.Is(err, tt.expectedErr) && !strings.Contains(err.Error(), tt.expectedErr.Error()) {
					t.Errorf("expected error to contain/be %q, got %q", tt.expectedErr, err.Error())
				}
			} else if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			// Assert Stdout
			if tt.expectedStdout != "" {
				if !strings.Contains(stdout.String(), tt.expectedStdout) {
					t.Errorf("expected stdout to contain %q, got %q", tt.expectedStdout, stdout.String())
				}
			}

			// Assert Stderr
			if tt.expectedStderr != "" {
				if !strings.Contains(stderr.String(), tt.expectedStderr) {
					t.Errorf("expected stderr to contain %q, got %q", tt.expectedStderr, stderr.String())
				}
			}
		})
	}
}