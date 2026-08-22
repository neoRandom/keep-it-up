# Step 05: Add Read Endpoints with Access Control

## Git Commit Plan
- **Title:** `feat(http): add read endpoints with access control`
- **Body:** Implements the three read endpoints required by TODO 1 under the
  authenticated `api` group, each enforcing TODO 3 access control:
  - `GET /api/games` → list games the player has access to.
  - `GET /api/shared?gameId=` → current shared state of a game.
  - `GET /api/interactions?gameId=&limit=` → latest interactions (default limit 20).

  All use the logged-in player's ID extracted from the JWT claims. Read endpoints
  call `AccessManagement.CheckPlayerAccess(gameId, playerId)` first; a missing
  access returns `404` (per spec, not `403`) to avoid revealing game existence.

## Task Description
- **What needs to be done:**
  1. In `internal/infrastructure/driver/httpadapter/http.go`:
     - Add to `Deps`:
       ```go
       Fetch  port.DataFetching
       Access port.AccessManagement
       ```
     - Add response DTOs (unexported or exported) whose JSON tags match the spec:
       ```go
       type sharedDataDTO struct {
           GameID       int64      `json:"gameId"`
           Status       string     `json:"status"`
           Valid        *bool      `json:"valid"`
           DeadlineAt   *time.Time `json:"deadlineAt"`
           LastSavedAt  *time.Time `json:"lastSavedAt"`
           LastPausedAt *time.Time `json:"lastPausedAt"`
       }
       type gameDTO struct { ID int64 `json:"id"`; Name string `json:"name"` }
       type interactionDTO struct {
           ID         int64      `json:"id"`
           GameID     int64      `json:"gameId"`
           PlayerID   *int64     `json:"playerId"`
           Action     string     `json:"action"`
           OccurredAt string     `json:"occurredAt"`
           SavedBy    *int64     `json:"savedBy"`
       }
       ```
     - Add helpers:
       ```go
       // playerIDFromContext returns the authenticated player ID from JWT claims.
       func (h *HTTPAdapter) playerIDFromContext(c *echo.Context) (int64, bool)

       // requireAccess enforces CheckPlayerAccess; returns 404 when no access.
       func (h *HTTPAdapter) requireAccess(c *echo.Context, gameID int64) bool
       ```
     - Register routes in `routes(e)` under the protected group:
       ```go
       api.GET("/games", h.handleListGames)
       api.GET("/shared", h.handleGetShared)
       api.GET("/interactions", h.handleListInteractions)
       ```
     - Implement the three handlers as methods on `HTTPAdapter`.
  2. Handler details:
     - `handleListGames`: `playerID, ok := h.playerIDFromContext(c)`; call
       `h.d.Fetch.ListPlayerGames(ctx, playerID)`; `200` with `[]{gameDTO}`.
     - `handleGetShared`: parse `gameId` query param (int64); if invalid → `400`;
       `requireAccess`; call `h.d.Fetch.GetSharedData`; map `*model.SharedData` to
       `sharedDataDTO`; `200`. On error → `500`.
     - `handleListInteractions`: parse `gameId` (required) and `limit` (optional,
       default `20`, per spec `minimum: 1`; reject values `< 1` with `400`);
       `requireAccess`; call `h.d.Fetch.ListInteractions`; map
       `[]database.Interaction` (handling `sql.NullInt64` → `*int64`) to
       `[]{interactionDTO}`; `200`.
- **Why it needs to be done:**
  TODO 1 wants the read-only surface of the API exposed. Without it, clients can
  only authenticate via HTTP and must use the CLI for reads. TODO 3 adds the
  security requirement: the `access` table is the link between player and game, and
  only players granted access may read a game's data. The spec's `404 GameNotFound`
  description ("Game not found **or inaccessible**") mandates that an inaccessible
  game be indistinguishable from a non-existent one, hence `404` rather than `403`.
  The player ID must come from the JWT `UserID` claim so the request is attributed
  to the authenticated actor, not an arbitrary/query parameter.

## Architecture & Testing
- **Invariants:**
  - Every read endpoint extracts the actor from JWT claims (`JwtPlayerClaims.UserID`).
  - `GET /shared` and `GET /interactions` return `404` when the player lacks access.
  - `GET /games` returns only the games the authenticated player has access to (the
    use case already filters by `access` join).
  - JSON field names match the spec (`gameId`, `status`, `valid`, `deadlineAt`,
    `lastSavedAt`, `lastPausedAt`, `playerId`, `occurredAt`, `savedBy`).
  - No `database.Null*` types are serialized directly.
- **Testing Strategy:**
  Handler unit tests in Step 08 cover: 200 read responses with correctly-typed
  JSON, 404 on denied access, 400 on bad `gameId`/`limit`, 401 without a valid
  cookie. No DB is used; fakes implement `port.DataFetching` and
  `port.AccessManagement`.
- **Verification:**
  ```bash
  go build ./...
  ```

## Expected Outcome
- **State of the project:**
  The authenticated API exposes `GET /api/games`, `GET /api/shared`, and
  `GET /api/interactions`, all access-controlled and returning spec-compliant JSON.
  The `Deps` struct now includes `Fetch` and `Access` (committed but not yet wired
  in `cmd/server/main.go`, which still compiles because the fields are additive).