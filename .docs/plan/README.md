# Execution Plan — TODOs 1, 3 & 4 (HTTP API, Access Control, Spec Alignment)

## Overarching Goal

Complete the HTTP API so the backend is fully usable over the network, matching
the contract in `api/openapi.yaml`:

- **TODO 1** — Expose all documented endpoints (`GET /games`, `GET /shared`,
  `GET /interactions`, `POST /save`, `POST /play`, `POST /pause`) by wiring the
  already-implemented `DataFetching` and `GameCommands` use cases into the HTTP
  adapter.
- **TODO 3** — Enforce access control on every authenticated endpoint via
  `AccessManagement.CheckPlayerAccess`, returning `404` (not `403`) when a player
  has no access, per the spec.
- **TODO 4** — Align the `login` handler and cookie/JWT lifetime with the
  OpenAPI spec (JSON body, `session` cookie name, shared 72h lifetime, `204` with
  no body).

The outcome is a clean, modular, spec-conformant HTTP layer with unit tests that
isolate handler logic via injected fakes, and integration tests that verify the
core domain invariants (state machine) against a real SQLite database.

## Global Invariants (must never be broken)

1. **Each atomic step leaves the repo compilable and testable.** Use
   `go build ./...` after every step before committing.
2. **No changes to TODOs 5, 6, or 7.** This plan only addresses TODOs 1, 3, 4 and
   the minimal tests supporting them. Do not add/remove DataFetching operations,
   do not clean up `config/` or `interactions.sql`, do not `sqlc` regenerate.
3. **Access control is enforced on every authenticated endpoint.** A player with
   no access must receive `404`, never `403`, to avoid leaking game existence.
4. **The spec (`api/openapi.yaml`) is the contract.** Do not change endpoint
   paths, query params, status codes, or the `session` cookie name. Do not alter
   the spec file.
5. **Responses match the spec field names.** Wire the domain DTOs to lowercase
   JSON keys (`gameId`, `status`, `valid`, `deadlineAt`, `lastSavedAt`,
   `lastPausedAt`, `playerId`, `occurredAt`, `savedBy`). Never serialize
   `database.Null*` types directly.
6. **Cookie and JWT share one lifetime.** Use the single `constant.SessionLifetime`
   constant for both the login cookie expiry and the JWT `exp` claim.
7. **`204 No Content` responses have no body.** Login, save, play, pause all
   return `c.NoContent(http.StatusNoContent)`.
8. **Keep domain layer clean.** All API serialization lives in
   `httpadapter` DTO structs; do not add JSON tags to `model` or `database` types
   unless strictly necessary for this plan.
9. **Unit vs integration separation.**
   - Unit tests (`httpadapter`, `driven`) use hand-written fakes/mocks, no DB.
   - Integration tests (`usecase`) use the real SQLite in-memory DB via
     `newTestDB(t)` (Goose migrations are the schema source of truth).

## Execution Order (atomic steps)

| # | Title | Type |
|---|-------|------|
| 01 | Add shared session lifetime constant | `feat(constant)` |
| 02 | Use shared lifetime in JWT generator | `refactor(driven)` |
| 03 | Extract testable router setup in httpadapter | `refactor(http)` |
| 04 | Align login handler with API spec | `fix(http)` |
| 05 | Add read endpoints with access control | `feat(http)` |
| 06 | Add command endpoints with access control | `feat(http)` |
| 07 | Wire new use cases in server main | `feat(server)` |
| 08 | Add HTTP handler unit tests | `test(http)` |
| 09 | Add state machine integration tests | `test(usecase)` |
| 10 | Add JWT generator unit test | `test(driven)` |

After every step, `go build ./...` must succeed. After any step that adds or
modifies tests (08–10), `go test <affected packages>/...` must pass.

## General Commands

```bash
# Build everything (sanity check after every step)
go build ./...

# Vet the whole module
go vet ./...

# Run the full test suite
go test ./...

# Run a single package's tests
go test ./internal/infrastructure/driver/httpadapter/...
go test ./internal/application/usecase/...
go test ./internal/infrastructure/driven/...

# Recommended commit flow (one commit per step)
git add -A && git commit -m "<standardized title>" -m "<body>"
```

## Step File Index

- [Step 01: Shared session lifetime constant](step_01_shared_session_lifetime.md)
- [Step 02: Use shared lifetime in JWT generator](step_02_use_shared_lifetime_in_jwt.md)
- [Step 03: Extract testable router setup](step_03_extract_testable_router.md)
- [Step 04: Align login handler with API spec](step_04_align_login_handler.md)
- [Step 05: Add read endpoints with access control](step_05_read_endpoints.md)
- [Step 06: Add command endpoints with access control](step_06_command_endpoints.md)
- [Step 07: Wire new use cases in server main](step_07_wire_server_main.md)
- [Step 08: HTTP handler unit tests](step_08_http_unit_tests.md)
- [Step 09: State machine integration tests](step_09_state_machine_integration_tests.md)
- [Step 10: JWT generator unit test](step_10_jwt_generator_test.md)