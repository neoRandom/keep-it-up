# Step 08: HTTP Handler Unit Tests

## Git Commit Plan
- **Title:** `test(http): add handler unit tests`
- **Body:** Adds `internal/infrastructure/driver/httpadapter/http_test.go`
  exercising all HTTP endpoints with injected fakes (no DB). Covers login
  (JSON body, `session` cookie, `204`), the read endpoints (200, 404 access
  denied, 400 bad params, 401 no cookie), and the command endpoints (204
  success, 404 access denied, 400 bad body, 409 state-machine conflict).
  Uses Echo test helpers via the extracted `routes` (or a test helper that
  builds an `*echo.Echo` from the adapter) plus hand-written fakes that
  implement the `port` interfaces.

## Task Description
- **What needs to be done:**
  1. Create `internal/infrastructure/driver/httpadapter/http_test.go`
     (package `httpadapter` or `httpadapter_test`; prefer internal for access to
     `routes`/`New`).
  2. Add fakes satisfying the ports used by `Deps`:
     - `fakeAuth` implementing `port.Authentication` (`LoginPlayer`,
       `CheckPlayerPassword`).
     - `fakeFetch` implementing `port.DataFetching` (always record calls;
       return canned `[]database.Game`, `*model.SharedData`,
       `[]database.Interaction` or configured errors).
     - `fakeCommands` implementing `port.GameCommands` (record calls; return
       configured errors, e.g. a state-machine error).
     - `fakeAccess` implementing `port.AccessManagement` (return configured
       `(bool, error)`).
  3. Add a test helper that builds a ready router:
     ```go
     func newTestRouter(t *testing.T, d Deps) *echo.Echo {
         t.Helper()
         adapter := New(":0", "test-secret", &fakeTime{}, d)
         e := echo.New()
         adapter.routes(e)
         return e
     }
     ```
     (A `fakeTime` implementing `port.TimeProvider` is needed for the login
     cookie path.)
  4. Write table-driven tests per endpoint:
     - **Login**: JSON success → `204`, `Set-Cookie` contains `session`; malformed
       JSON → `400`; empty creds → `400`; wrong creds → `401`.
     - **Auth guard**: hitting any protected endpoint without a cookie → `401`.
     - **GET /api/games**: `200` with the games array.
     - **GET /api/shared**: with access `200` + correct JSON keys; no access
       `404`; bad `gameId` → `400`.
     - **GET /api/interactions**: `200` + JSON array; no access `404`; bad
       `gameId`/`limit` → `400`.
     - **POST /api/save**: `204`; no access `404`; invalid body/duration → `400`;
       fakeCommands returns state-machine error → `409`.
     - **POST /api/play**, **POST /api/pause**: `204`; no access `404`; conflict
       error → `409`.
  5. For protected routes, the JWT middleware needs a valid token. Generate one via
     the real `JwtTokenGenerator` with a known secret and time, and set it as the
     `session` cookie on the request so the middleware passes and `playerIDFromContext`
     resolves.
- **Why it needs to be done:**
  The HTTP layer is the public API — a regression breaks every client. These tests
  lock in the spec contract (status codes, `session` cookie, JSON key names) and
  the access-control rule (denied → `404`) with fast, hermetic unit tests. They also
  prove the extracted `routes` seam (Step 03) works and that handlers correctly
  translate domain results into HTTP responses.

## Architecture & Testing
- **Invariants:**
  - No real DB or network sockets; all deps are fakes.
  - Protected endpoints return `401` without a valid `session` cookie.
  - Denied access returns `404`, never `403`.
  - Success responses use the exact spec status codes and JSON key names.
- **Testing Strategy:**
  Table-driven unit tests with fakes; assert status codes, `Set-Cookie` name, and
  decoded JSON bodies. This is the primary unit-test layer for the HTTP handlers.
- **Verification:**
  ```bash
  go test ./internal/infrastructure/driver/httpadapter/... -count=1
  ```

## Expected Outcome
- **State of the project:**
  `httpadapter` has comprehensive unit tests covering login, read, and command
  endpoints, including access-control and error mappings. The package's tests pass.