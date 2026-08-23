package mapping

import (
	"database/sql"
	"fmt"
	"time"

	"keep-it-up/internal/core/model"
	"keep-it-up/internal/infrastructure/constant"
	"keep-it-up/internal/infrastructure/database"
)

// ToDomainInteraction maps a stored interaction to the domain Interaction,
// parsing the persisted RFC3339 timestamp into a time.Time.
func ToDomainInteraction(i database.Interaction) (model.Interaction, error) {
	occurredAt, err := time.Parse(constant.DBDatetimeFormat, i.OccurredAt)
	if err != nil {
		return model.Interaction{}, fmt.Errorf(
			"map interaction %d: parse occurred_at: %w", i.ID, err,
		)
	}
	return model.Interaction{
		ID:         i.ID,
		GameID:     i.GameID,
		PlayerID:   nullableInt64(i.PlayerID),
		Action:     i.Action,
		OccurredAt: occurredAt,
		SavedBy:    nullableInt64(i.SavedBy),
	}, nil
}

// ToDomainInteractions maps a slice of stored interactions to domain
// Interactions. It fails-fast on the first unparseable timestamp and returns a nil slice (no partial results).
func ToDomainInteractions(is []database.Interaction) ([]model.Interaction, error) {
	out := make([]model.Interaction, 0, len(is))
	for _, i := range is {
		d, err := ToDomainInteraction(i)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

// nullableInt64 maps a database.NullInt64 to *int64, nil when invalid.
func nullableInt64(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	x := v.Int64
	return &x
}
