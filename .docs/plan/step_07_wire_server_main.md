# Step 07: Wire New Use Cases in Server Main

## Git Commit Plan
- **Title:** `feat(server): wire DataFetching, GameCommands, AccessManagement`
- **Body:** Constructs the `DataFetching`, `GameCommands`, and `AccessManagement`
  use cases in `cmd/server/main.go` and injects them into `httpadapter.Deps`
  alongside the existing `Auth`. This activates the read and command endpoints
  added in Steps 05 and 06 in the real server.

## Task Description
- **What needs to be done:**
  1. Open `cmd/server/main.go`.
  2. After `q := database.New(sqlDB)` and before building `deps`, construct the use
     cases. Ensure `timeProvider` is created once and reused by both
     `NewGameCommands` and the existing `JwtTokenGenerator`:
     ```go
     fetching := usecase.NewDataFetching(q)
     commands := usecase.NewGameCommands(q, timeProvider)
     access := usecase.NewAccessManagement(q)
     ```
  3. Populate the `httpadapter.Deps` with all four fields:
     ```go
     deps := httpadapter.Deps{
         Auth:     usecase.NewAuthentication(q, &driven.JwtTokenGenerator{...}),
         Fetch:    fetching,
         Commands: commands,
         Access:   access,
     }
     ```
- **Why it needs to be done:**
  Steps 05/06 added the endpoints and the `Deps` fields, but the real server builds
  its `Deps` with only `Auth`. Until the use cases are wired here, the HTTP server
  would have nil `Fetch`/`Commands`/`Access`, causing nil-pointer behavior at
  runtime when those endpoints are hit. This step is the composition root that
  connects the infrastructure adapter to the application layer, completing TODO 1's
  "wire like Auth is wired" instruction.

## Architecture & Testing
- **Invariants:**
  - `timeProvider` is a single shared instance passed to both `NewGameCommands` and
    the JWT generator.
  - All four `Deps` fields (`Auth`, `Fetch`, `Commands`, `Access`) are non-nil.
- **Testing Strategy:**
  Composition-root change; verified by `go build` and `go vet`. HTTP handler
  behavior is exercised via fakes in Step 08, not through this wiring.
- **Verification:**
  ```bash
  go build ./...
  go vet ./cmd/server/...
  ```

## Expected Outcome
- **State of the project:**
  `cmd/server` assembles all four use cases and injects them into the HTTP adapter,
  so the real server serves the complete, access-controlled API surface.