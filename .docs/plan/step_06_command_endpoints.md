# Step 06: Add Command Endpoints with Access Control

## Git Commit Plan
- **Title:** `feat(http): add command endpoints with access control`
- **Body:** Implements the three write endpoints required by TODO 1 under the
  authenticated `api` group, each enforcing TODO 3 access control:
  - `POST /api/save?gameId=` + JSON body `{duration}` → add a save interaction.
  - `POST /api/play?gameId=` → start or resume the game.
  - `POST /api/pause?gameId=` → pause the game.

  Commands call `AccessManagement.CheckPlayerAccess(gameId, playerId)` first; a
  missing access returns `404` (per spec, not `403`). Domain state-machine
  violations (e.g. save-while-paused) map to `409 Conflict`.

## Task Description
- **What needs to be done:**
  1. In `internal/infrastructure/driver/httpadapter/http.go`:
     - Add to `Deps`:
       ```go
       Commands port.GameCommands
       ```
     - Add a request DTO for save:
       ```go
       type saveRequest struct {
           Duration int64 `json:"duration"`
       }
       ```
     - Register routes in `routes(e)` under the protected group:
       ```go
       api.POST("/save", h.handleSave)
       api.POST("/play", h.handleResume)
       api.POST("/pause", h.handlePause)
       ```
     - Implement the three handlers as methods on `HTTPAdapter`, plus a small
       helper `conflictStatusFromErr(err error) bool` that reports `409` when the
       save/play/pause operation is rejected by the domain state machine.
  2. Handler details:
     - `handleSave`: parse `gameId` query param (required; invalid → `400`);
       `requireAccess`; bind JSON `{duration}` (invalid/missing → `400`; `duration<1`
       → `400`); call `h.d.Commands.SaveGame(ctx, gameID, playerID, duration)`;
       on success `c.NoContent(http.StatusNoContent)`; on error map
       `409` (state machine) else `500`.
     - `handleResume`: parse `gameId`; `requireAccess`; call
       `h.d.Commands.ResumeGame(ctx, gameID, playerID)`; success → `204`;
       error → `409`/`500`.
     - `handlePause`: parse `gameId`; `requireAccess`; call
       `h.d.Commands.PauseGame(ctx, gameID, playerID)`; success → `204`;
       error → `409`/`500`.
  3. State-machine error detection: the sqlite trigger raises errors with messages
     like `"cannot save while paused"`, `"cannot pause: game is not currently
     playing"`, `"cannot resume: game is not currently paused"`. Implement
     `conflictStatusFromErr` by checking these substrings (or define sentinel errors
     in the use case if preferred). Keep the detection in the HTTP layer only.
- **Why it needs to be done:**
  TODO 1 requires the write surface of the API. Without `/save`, `/play`, and
  `/pause`, clients cannot drive the game state machine over HTTP. TODO 3 requires
  that all of these verify the player's `access` before acting — otherwise any
  authenticated user could mutate a game they don't belong to. The spec maps
  invalid transitions to `409` ("Game is not currently playing" / "Game is already
  playing"), which corresponds to the DB trigger rejecting the insert. Returning
  `404` instead of `403` for missing access keeps inaccessible games
  indistinguishable from non-existent ones, exactly as the spec's `GameNotFound`
  response describes.

## Architecture & Testing
- **Invariants:**
  - Every command endpoint extracts the actor from JWT claims.
  - Access is checked before any mutation; denied access → `404`.
  - Success is an empty `204` (no body).
  - State-machine violations → `409`.
  - Malformed body or invalid `gameId` → `400`.
- **Testing Strategy:**
  Handler unit tests in Step 08 cover: `204` on success, `404` on denied access,
  `400` on bad `gameId`/`duration`/JSON, `409` on state-machine conflict, and `401`
  without a cookie. Fakes implement `port.GameCommands` and `port.AccessManagement`.
- **Verification:**
  ```bash
  go build ./...
  ```

## Expected Outcome
- **State of the project:**
  The authenticated API exposes `POST /api/save`, `POST /api/play`, and
  `POST /api/pause`, all access-controlled, returning `204` on success and
  `409` on invalid transitions. The `Deps` struct now includes `Commands`.