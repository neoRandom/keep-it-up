package service

import (
	"fmt"
	"time"

	"keep-it-up/internal/core/model"
	"keep-it-up/internal/infrastructure/constant"
	"keep-it-up/internal/infrastructure/database"
)

func ComputeValid(s *model.SharedData, now time.Time) {
	if s.DeadlineAt == nil {
		s.Valid = nil
		return
	}

	valid := now.Before(*s.DeadlineAt)

	// A paused game is still valid as long as the pause happened after the
	// last save and before the deadline. While paused, the clock is frozen,
	// so the deadline is not consumed by the elapsed pause time.
	if s.Status == model.Paused &&
		s.LastPausedAt != nil &&
		s.LastSavedAt != nil &&
		s.LastPausedAt.Before(*s.DeadlineAt) &&
		s.LastPausedAt.After(*s.LastSavedAt) {
		valid = true
	}

	s.Valid = &valid
}

// BuildSharedData assembles the current SharedData from a game's interactions.
// `now` is used to compute the time-dependent "valid" flag, so the returned
// value always carries a Valid consistent with its deadline.
func BuildSharedData(
	gameId int64, interactions []database.Interaction, now time.Time,
) (*model.SharedData, error) {
	data := &model.SharedData{
		GameID: gameId,
		Status: model.NotStarted,
	}

	if len(interactions) == 0 {
		return data, nil
	}

	var prevOccurredAt time.Time

	for _, ia := range interactions {
		if ia.GameID != data.GameID {
			return nil, fmt.Errorf(
				"build shared data: interaction %d belongs to game %d, expected %d",
				ia.ID, ia.GameID, data.GameID,
			)
		}

		occurredAt, err := time.Parse(constant.DBDatetimeFormat, ia.OccurredAt)
		if err != nil {
			return nil, fmt.Errorf(
				"build shared data: interaction %d: parse occurred_at: %w",
				ia.ID, err,
			)
		}
		if !prevOccurredAt.IsZero() && occurredAt.Before(prevOccurredAt) {
			return nil, fmt.Errorf(
				"build shared data: interaction %d: occurred_at out of order",
				ia.ID,
			)
		}
		prevOccurredAt = occurredAt

		switch ia.Action {
		case "saved":
			if data.Status == model.Paused {
				return nil, fmt.Errorf(
					"build shared data: interaction %d: cannot save while paused",
					ia.ID,
				)
			}
			if !ia.SavedBy.Valid || ia.SavedBy.Int64 <= 0 {
				return nil, fmt.Errorf(
					"build shared data: interaction %d: invalid saved_by",
					ia.ID,
				)
			}
			extension := time.Duration(ia.SavedBy.Int64) * time.Second

			if data.Status == model.NotStarted {
				deadline := occurredAt.Add(extension)
				data.DeadlineAt = &deadline
				data.Status = model.Playing
			} else {
				deadline := data.DeadlineAt.Add(extension)
				data.DeadlineAt = &deadline
			}
			data.LastSavedAt = &occurredAt

		case "paused":
			if data.Status != model.Playing {
				return nil, fmt.Errorf(
					"build shared data: interaction %d: cannot pause from status %q",
					ia.ID, data.Status,
				)
			}
			data.Status = model.Paused
			data.LastPausedAt = &occurredAt

		case "resumed":
			if data.Status != model.Paused {
				return nil, fmt.Errorf(
					"build shared data: interaction %d: cannot resume from status %q",
					ia.ID, data.Status,
				)
			}
			deadline := data.DeadlineAt.Add(occurredAt.Sub(*data.LastPausedAt))
			data.DeadlineAt = &deadline
			data.Status = model.Playing
			data.LastPausedAt = nil

		default:
			return nil, fmt.Errorf(
				"build shared data: interaction %d: unknown action %q",
				ia.ID, ia.Action,
			)
		}
	}

	ComputeValid(data, now)
	return data, nil
}
