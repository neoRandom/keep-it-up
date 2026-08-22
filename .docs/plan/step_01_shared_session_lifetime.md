# Step 01: Shared Session Lifetime Constant

## Git Commit Plan
- **Title:** `feat(constant): add shared session lifetime constant`
- **Body:** Introduces a single `SessionLifetime` constant (72h) that will be shared
  by both the HTTP login cookie expiry and the JWT `exp` claim, so the two can never
  drift apart. This is the foundation for TODO 4 ("Cookie vs JWT lifetime").

## Task Description
- **What needs to be done:**
  1. Open `internal/infrastructure/constant/constant.go`.
  2. Define a new exported constant `SessionLifetime` of type `time.Duration` with
     value `72 * time.Hour`.
- **Why it needs to be done:**
  TODO 4 requires that the cookie expiry (currently 24h in `http.go`) and the JWT
  lifetime (currently 72h in `jwt_token_generator.go`) be identical so a browser
  session cookie never expires before the token it carries. A single shared constant
  removes the risk of future drift and documents the intended session length in one
  authoritative place. It will be used by later steps; this step only adds the
  constant so every subsequent step can reference it.

## Architecture & Testing
- **Invariants:**
  - `constant.SessionLifetime` must equal exactly 72 hours.
  - No behavior changes yet; the constant is added but not yet consumed.
- **Testing Strategy:**
  No functional tests needed for a constant. The constant will be exercised by the
  JWT generator test (Step 10) and the HTTP handler tests (Step 08).
- **Verification:**
  ```bash
  go build ./...
  ```

## Expected Outcome
- **State of the project:**
  The `constant` package exposes `SessionLifetime = 72 * time.Hour`. The rest of the
  codebase is unchanged and still compiles.