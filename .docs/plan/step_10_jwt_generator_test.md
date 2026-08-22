# Step 10: JWT Generator Unit Test

## Git Commit Plan
- **Title:** `test(driven): add JWT generator unit test`
- **Body:** Adds `internal/infrastructure/driven/jwt_token_generator_test.go`
  verifying that `JwtTokenGenerator.GenerateToken` produces a signed token that:
  - verifies with the same secret,
  - uses HMAC-SHA256,
  - carries the player's `UserID` and `Username` claims,
  - has an `exp` that equals `now + constant.SessionLifetime`

  Uses a fake `TimeProvider` returning a fixed time.

## Task Description
- **What needs to be done:**
  1. Create `internal/infrastructure/driven/jwt_token_generator_test.go`
     (package `driven` or `driven_test`).
  2. Implement a `fixedClock` fake implementing `port.TimeProvider` returning a
     fixed `time.Time` (e.g. `time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)`).
  3. Write tests:
     - `TestGenerateToken_ValidClaims`: generate a token for a
       `database.Player{ID: 42, Username: "neo"}`, parse it back with
       `jwt.ParseWithClaims` using `&model.JwtPlayerClaims{}` and the same secret;
       assert `UserID == 42`, `Username == "neo"`, and the signing method is
       `jwt.SigningMethodHS256`.
     - `TestGenerateToken_ExpiryMatchesSessionLifetime`: assert the parsed token's
       `RegisteredClaims.ExpiresAt` is exactly `fixedTime.Add(constant.SessionLifetime)`.
     - `TestGenerateToken_Errors`: empty secret and nil `TimeProvider` return errors.
- **Why it needs to be done:**
  The JWT generator controls authentication security, and it is the source of the
  token stored in the `session` cookie. Verifying that the token carries the right
  claims and that its expiry matches `constant.SessionLifetime` (the SAME constant
  the login cookie uses, per Step 04) locks in TODO 4's lifetime-alignment invariant
  and guards the auth mechanism. It also documents the token format for future
  maintainers.

## Architecture & Testing
- **Invariants:**
  - Token verifies with the same secret and fails with a different one.
  - `exp` claim equals `now + constant.SessionLifetime`.
  - `UserID`/`Username` claims round-trip exactly.
- **Testing Strategy:**
  Unit test with a fake clock; no DB, no network. Parse the token back with the
  JWT library to assert claims and expiry. This is the third unit-test layer
  (alongside HTTP handlers and use case integrations).
- **Verification:**
  ```bash
  go test ./internal/infrastructure/driven/... -count=1
  ```

## Expected Outcome
- **State of the project:**
  `internal/infrastructure/driven` has unit tests asserting the JWT token's claims,
  expiry (aligned with `constant.SessionLifetime`), and error handling. The package's
  tests pass.