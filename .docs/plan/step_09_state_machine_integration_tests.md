# Step 09: State Machine Integration Tests

## Git Commit Plan
- **Title:** `test(usecase): add game command state-machine integration tests`
- **Body:** Adds integration tests in `internal/application/usecase/game_commands_test.go`
  that exercise the domain state machine against a real SQLite in-memory DB (via
  the existing `newTestDB(t)` helper + Goose migrations). Verifies the core domain
  invariants the HTTP handlers depend on: save-while-paused is rejected,
  pause-without-playing is rejected, resume-without-paused is rejected, and a valid
  save → pause → resume sequence succeeds. Also covers invalid input boundaries
  (gameId/playerId/duration).

## Task Description
- **What needs to be done:**
  1. Create `internal/application/usecase/game_commands_test.go`
     (package `usecase`).
  2. Implement a `fixedClock` implementing `port.TimeProvider` that returns a
     controllable, monotonically increasing `time.Time` (e.g. `time.Date(2026, 8,
     22, 12, 0, 0, 0, time.UTC)`). `GameCommands` requires `tp` for the
     `occurred_at` timestamp; use a fixed time so ordering is deterministic and the
     state-machine trigger accepts inserts (timestamps must be non-decreasing per
     game).
  3. Write integration tests:
     - `TestGameCommands_ValidSaveResumePauseSequence`: `SaveGame` → `PauseGame` →
       `ResumeGame` all succeed; assert the resulting interactions via
       `DataFetching.ListInteractions` (or direct query).
     - `TestGameCommands_SaveWhilePausedRejected`: `SaveGame`, `PauseGame`, then a
       second `SaveGame` returns an error containing `"cannot save while paused"`.
     - `TestGameCommands_PauseWhenNotPlayingRejected`: `PauseGame` as the first
       interaction returns an error containing `"cannot pause"`.
     - `TestGameCommands_ResumeWhenNotPausedRejected`: `ResumeGame` as the first
       interaction returns an error containing `"cannot resume"`.
     - `TestGameCommands_InvalidInputs`: zero/negative `gameId`, zero/negative
       `playerId`, and `duration < 1` return errors.
     - `TestGameCommands_NilDependencies`: nil `q` and nil `tp` return errors.
- **Why it needs to be done:**
  The HTTP layer relies on these domain invariants: the spec's `409` responses for
  `/save`, `/play`, `/pause` are only meaningful if the underlying state machine
  enforces them. The state machine lives in the SQLite trigger
  (`trg_interactions_state_machine`) and `BuildSharedData`. Integration tests
  against the real migrated DB lock in these invariants and are the second half of
  the "unit vs integration" split: here we verify end-to-end persistence and
  trigger behavior, not just handler routing.

## Architecture & Testing
- **Invariants:**
  - `SaveGame` requires a positive `duration` and a valid game/player.
  - `PauseGame` only succeeds when the game is currently playing (last action
    `saved` or `resumed`).
  - `ResumeGame` only succeeds when the game is currently paused.
  - `SaveGame` is rejected while paused.
  - The trigger's error messages are surfaced by the use case.
- **Testing Strategy:**
  Integration: real SQLite via `newTestDB(t)` and a fixed `TimeProvider`. Assert
  error substrings matching the trigger messages and the success/failure of the
  sequence. These couple to the migrated schema, so they are true end-to-end (DB)
  tests for domain logic.
- **Verification:**
  ```bash
  go test ./internal/application/usecase/... -count=1
  ```

## Expected Outcome
- **State of the project:**
  `internal/application/usecase` has integration tests locking down the game state
  machine invariants and invalid-input boundaries. The use case package's tests pass.