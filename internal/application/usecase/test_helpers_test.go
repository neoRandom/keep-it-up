package usecase

import (
	"testing"

	"keep-it-up/internal/infrastructure/database"
	"keep-it-up/internal/testutil"
)

// newTestDB is a thin wrapper over testutil.NewTestDB so existing call sites
// stay concise; the shared schema/migration setup lives in internal/testutil.
func newTestDB(t *testing.T) *database.Queries {
	return testutil.NewTestDB(t)
}
