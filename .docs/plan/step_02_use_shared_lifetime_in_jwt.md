# Step 02: Use Shared Lifetime in JWT Generator

## Git Commit Plan
- **Title:** `refactor(driven): use shared SessionLifetime for JWT expiry`
- **Body:** Replaces the hardcoded `time.Hour * 72` in `JwtTokenGenerator` with the
  shared `constant.SessionLifetime`. Behavior is identical (72h), but the expiry now
  derives from a single source of truth that the HTTP cookie will also use.

## Task Description
- **What needs to be done:**
  1. Open `internal/infrastructure/driven/jwt_token_generator.go`.
  2. Import `keep-it-up/internal/infrastructure/constant`.
  3. Replace `t.Add(time.Hour * 72)` with `t.Add(constant.SessionLifetime)`.
  4. Remove the now-unused `time` import only if `time` is no longer referenced
     (it is still needed for `time.Time` from the `TimeProvider` return, so keep it).
- **Why it needs to be done:**
  TODO 4 ("Cookie vs JWT lifetime") requires the JWT's `exp` claim and the session
  cookie's expiry to match. Centralizing the lifetime in `constant.SessionLifetime`
  (added in Step 01) guarantees the two stay in lockstep and removes magic-number
  literals from the auth-critical infrastructure code.

## Architecture & Testing
- **Invariants:**
  - JWT expiry remains exactly 72 hours from `TimeProvider.Time()`.
  - No signature/claims change.
- **Testing Strategy:**
  Behavior is unchanged; existing tests should continue to pass. A dedicated unit
  test asserting the token's expiry equals `now + constant.SessionLifetime` is added
  in Step 10.
- **Verification:**
  ```bash
  go build ./...
  go test ./internal/infrastructure/driven/...
  ```

## Expected Outcome
- **State of the project:**
  `JwtTokenGenerator` issues tokens expiring exactly `constant.SessionLifetime`
  after the current time. The package still compiles and its tests (if any) pass.