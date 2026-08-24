package usecase

import (
	"testing"

	coremodel "keep-it-up/internal/core/model"
	"keep-it-up/internal/core/port"
	"keep-it-up/internal/infrastructure/database"
	"keep-it-up/internal/testutil"
)

// newTestDB is a thin wrapper over testutil.NewTestDB so existing call sites
// stay concise; the shared schema/migration setup lives in internal/testutil.
func newTestDB(t *testing.T) *database.Queries {
	return testutil.NewTestDB(t)
}

func mustNewGameManagement(t *testing.T, q *database.Queries) *GameManagement {
	t.Helper()
	uc, err := NewGameManagement(q)
	if err != nil {
		t.Fatalf("NewGameManagement: %v", err)
	}
	return uc
}

func mustNewAccessManagement(t *testing.T, q *database.Queries) *AccessManagement {
	t.Helper()
	uc, err := NewAccessManagement(q)
	if err != nil {
		t.Fatalf("NewAccessManagement: %v", err)
	}
	return uc
}

func mustNewDataFetching(t *testing.T, q *database.Queries, tp port.TimeProvider) *DataFetching {
	t.Helper()
	uc, err := NewDataFetching(q, tp)
	if err != nil {
		t.Fatalf("NewDataFetching: %v", err)
	}
	return uc
}

func mustNewGameCommands(t *testing.T, q *database.Queries, tp port.TimeProvider) *GameCommands {
	t.Helper()
	uc, err := NewGameCommands(q, tp)
	if err != nil {
		t.Fatalf("NewGameCommands: %v", err)
	}
	return uc
}

func mustNewAuthentication(t *testing.T, q *database.Queries, tg port.TokenGenerator) *Authentication {
	t.Helper()
	uc, err := NewAuthentication(q, tg)
	if err != nil {
		t.Fatalf("NewAuthentication: %v", err)
	}
	return uc
}

// stubTokenGenerator satisfies port.TokenGenerator so tests can construct
// Authentication with a non-nil tg (the constructor now requires one).
type stubTokenGenerator struct{}

func (stubTokenGenerator) GenerateToken(coremodel.Player) (string, error) {
	return "stub-token", nil
}

var stubTG port.TokenGenerator = stubTokenGenerator{}
