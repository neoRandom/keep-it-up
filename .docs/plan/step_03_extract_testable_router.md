# Step 03: Extract Testable Router Setup

## Git Commit Plan
- **Title:** `refactor(http): extract router setup into a testable method`
- **Body:** Moves route registration out of `Run(ctx)` into a dedicated
  `routes(e *echo.Echo)` method on `HTTPAdapter`. `Run` keeps the server lifecycle
  (listen, graceful shutdown) but delegates all handler/middleware wiring to
  `routes`. This makes the router unit-testable with Echo's test helpers (Echo +
  httptest) without opening a real listener, and it keeps `Run` focused on
  lifecycle concerns.

## Task Description
- **What needs to be done:**
  1. Open `internal/infrastructure/driver/httpadapter/http.go`.
  2. In `Run(ctx)`, after `e := echo.New()` and the global middleware
     (`RequestLogger`, `Recover`), call `h.routes(e)` instead of registering
     routes inline.
  3. Add a new unexported method:
     ```go
     func (h *HTTPAdapter) routes(e *echo.Echo) {
         // move the entire login handler + protected group + /test stub here
     }
     ```
  4. Keep the rest of `Run` (server construction, goroutine, select) unchanged.
  5. Do NOT yet add the new `Deps` fields or endpoints — those come in Steps 05/06.
     This step only relocates existing wiring verbatim.
- **Why it needs to be done:**
  Step 08 requires unit-testing HTTP handlers via Echo's `httptest` helpers. That
  is only possible if the router can be built as an `*echo.Echo` without starting a
  listening server. Extracting `routes` gives tests a seam: they can construct the
  adapter with fake `Deps` and call `routes(e)` (or a wrapping `NewHandler`) to
  exercise handlers. It also improves separation of concerns — wiring vs. lifecycle.

## Architecture & Testing
- **Invariants:**
  - `Run`'s observable behavior is unchanged (same middleware, same routes).
  - No new endpoints or dependency fields are introduced in this step.
- **Testing Strategy:**
  Pure refactor. No new tests here. The extracted router is exercised by
  Step 08's handler unit tests. Run the existing build/vet to confirm no
  behavioral regression.
- **Verification:**
  ```bash
  go build ./...
  go vet ./internal/infrastructure/driver/httpadapter/...
  ```

## Expected Outcome
- **State of the project:**
  `HTTPAdapter` has a `routes(*echo.Echo)` method containing all current route
  definitions; `Run(ctx)` calls it. The server starts and stops identically to
  before. The package compiles cleanly.