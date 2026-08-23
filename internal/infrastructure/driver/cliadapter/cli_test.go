package cliadapter_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"keep-it-up/internal/core/model"
	"keep-it-up/internal/infrastructure/database"
	"keep-it-up/internal/infrastructure/driver/cliadapter"
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

type mockFetch struct {
	ListPlayerGamesFunc      func(ctx context.Context, playerID int64) ([]database.Game, error)
	GetSharedDataFunc        func(ctx context.Context, gameID int64) (*model.SharedData, error)
	ListInteractionsFunc     func(ctx context.Context, gameID, limit, offset int64) ([]database.Interaction, error)
	ListPlayerInteractionsFunc func(ctx context.Context, gameID, playerID, limit, offset int64) ([]database.Interaction, error)
	FirstInteractionFunc     func(ctx context.Context, gameID int64) (*database.Interaction, error)
	LastInteractionFunc      func(ctx context.Context, gameID int64) (*database.Interaction, error)
}

func (m *mockFetch) ListPlayerGames(ctx context.Context, playerID int64) ([]database.Game, error) {
	return m.ListPlayerGamesFunc(ctx, playerID)
}
func (m *mockFetch) GetSharedData(ctx context.Context, gameID int64) (*model.SharedData, error) {
	return m.GetSharedDataFunc(ctx, gameID)
}
func (m *mockFetch) ListInteractions(ctx context.Context, gameID, limit, offset int64) ([]database.Interaction, error) {
	return m.ListInteractionsFunc(ctx, gameID, limit, offset)
}
func (m *mockFetch) ListPlayerInteractions(ctx context.Context, gameID, playerID, limit, offset int64) ([]database.Interaction, error) {
	return m.ListPlayerInteractionsFunc(ctx, gameID, playerID, limit, offset)
}
func (m *mockFetch) FirstInteraction(ctx context.Context, gameID int64) (*database.Interaction, error) {
	return m.FirstInteractionFunc(ctx, gameID)
}
func (m *mockFetch) LastInteraction(ctx context.Context, gameID int64) (*database.Interaction, error) {
	return m.LastInteractionFunc(ctx, gameID)
}

type mockCommands struct {
	SaveGameFunc   func(ctx context.Context, gameID, playerID int64, duration int64) error
	ResumeGameFunc func(ctx context.Context, gameID, playerID int64) error
	PauseGameFunc  func(ctx context.Context, gameID, playerID int64) error
}

func (m *mockCommands) SaveGame(ctx context.Context, gameID, playerID int64, duration int64) error {
	return m.SaveGameFunc(ctx, gameID, playerID, duration)
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
		setupMocks     func(d *cliadapter.Deps)
		expectedErr    error
		expectedStdout string
		expectedStderr string
	}{
		// Basic CLI checks
		{
			name:           "no arguments returns ErrNoCommand",
			args:           []string{},
			expectedErr:    cliadapter.ErrNoCommand,
			expectedStderr: "Usage:",
		},
		{
			name:           "help flag returns usage on stdout",
			args:           []string{"--help"},
			expectedErr:    nil,
			expectedStdout: "Usage:",
		},
		{
			name:           "unknown noun returns ErrUnknownCommand",
			args:           []string{"unknown-noun"},
			expectedErr:    cliadapter.ErrUnknownCommand,
			expectedStderr: "Usage:",
		},

		// Game subcommand tests
		{
			name:        "game without verb returns ErrNoSubcommand",
			args:        []string{"game"},
			expectedErr: cliadapter.ErrNoSubcommand,
		},
		{
			name: "game add success",
			args: []string{"game", "add", "MyGame"},
			setupMocks: func(d *cliadapter.Deps) {
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
			name:        "game add fails wrong arg count",
			args:        []string{"game", "add", "Too", "Many"},
			expectedErr: cliadapter.ErrWrongArgCount,
		},
		{
			name: "game update invalid id",
			args: []string{"game", "update", "not-an-id", "NewName"},
			// Will be wrapped in parseID error, so we just check if it returns an error
			expectedErr: errors.New("invalid id"),
		},

		// Command subcommand tests
		{
			name: "command save success",
			args: []string{"command", "save", "10", "20", "300"},
			setupMocks: func(d *cliadapter.Deps) {
				d.Commands = &mockCommands{
					SaveGameFunc: func(ctx context.Context, gameID, playerID int64, duration int64) error {
						if gameID != 10 || playerID != 20 {
							t.Errorf("unexpected IDs")
						}
						if duration != 300 {
							t.Errorf("expected duration 300, got %d", duration)
						}
						return nil
					},
				}
			},
			expectedErr:    nil,
			expectedStdout: "game 10 saved by player 20 for 300 sec\n",
		},
		{
			name:        "command save with invalid timestamp",
			args:        []string{"command", "save", "10", "20", "bad-time"},
			expectedErr: errors.New("invalid timestamp"),
		},
		{
			name: "command resume business error",
			args: []string{"command", "resume", "1", "2"},
			setupMocks: func(d *cliadapter.Deps) {
				d.Commands = &mockCommands{
					ResumeGameFunc: func(ctx context.Context, gameID, playerID int64) error {
						return dummyErr
					},
				}
			},
			expectedErr: dummyErr,
		},

		// Fetch interactions subcommand tests
		{
			name: "fetch interactions without offset defaults offset to 0",
			args: []string{"data", "interactions", "10", "5"},
			setupMocks: func(d *cliadapter.Deps) {
				d.Fetch = &mockFetch{
					ListInteractionsFunc: func(ctx context.Context, gameID, limit, offset int64) ([]database.Interaction, error) {
						if gameID != 10 || limit != 5 || offset != 0 {
							t.Errorf("unexpected args: gameID=%d limit=%d offset=%d", gameID, limit, offset)
						}
						return []database.Interaction{
							{ID: 1, GameID: 10, Action: "saved", OccurredAt: "2026-08-22T12:00:00Z"},
						}, nil
					},
				}
			},
			expectedErr:    nil,
			expectedStdout: "{ID:1",
		},
		{
			name: "fetch interactions with explicit offset",
			args: []string{"data", "interactions", "10", "5", "20"},
			setupMocks: func(d *cliadapter.Deps) {
				d.Fetch = &mockFetch{
					ListInteractionsFunc: func(ctx context.Context, gameID, limit, offset int64) ([]database.Interaction, error) {
						if gameID != 10 || limit != 5 || offset != 20 {
							t.Errorf("unexpected args: gameID=%d limit=%d offset=%d", gameID, limit, offset)
						}
						return nil, nil
					},
				}
			},
			expectedErr:    nil,
		},
		{
			name:        "fetch interactions with invalid offset",
			args:        []string{"data", "interactions", "10", "5", "bad-offset"},
			expectedErr: errors.New("invalid offset"),
		},
		{
			name:        "fetch interactions with too many args",
			args:        []string{"data", "interactions", "10", "5", "0", "extra"},
			expectedErr: cliadapter.ErrWrongArgCount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer

			// Construct dependencies
			deps := cliadapter.Deps{
				Stdout: &stdout,
				Stderr: &stderr,
			}

			// Apply test-specific mock setup
			if tt.setupMocks != nil {
				tt.setupMocks(&deps)
			}

			cliApp := cliadapter.New(deps)
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
