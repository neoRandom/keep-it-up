# Step 04: Align Login Handler with API Spec

## Git Commit Plan
- **Title:** `fix(http): align login handler with API spec`
- **Body:** Fixes the login handler's mismatches with `api/openapi.yaml`:
  - The spec declares an `application/json` body (`LoginRequest`); the handler
    currently reads `FormValue("username"/"password")`. Switch to decoding a JSON
    body and return `400` on malformed/missing fields.
  - The spec's security scheme names the cookie `session`; the handler previously
    used `access_token`. Rename the cookie constant to `session` and update the
    Echo JWT `TokenLookup` accordingly.
  - Cookie expiry was 24h while the JWT lasts 72h. Use `constant.SessionLifetime`
    (72h) for the cookie so session lifetime is predictable and matches the token.
  - A `204 No Content` response must not carry a body; replace the JSON body with
    `c.NoContent(http.StatusNoContent)`.

## Task Description
- **What needs to be done:**
  1. In `internal/infrastructure/driver/httpadapter/http.go`:
     - Rename `JWTTokenCookieName` to `"session"`. Keep the exported constant name
       or introduce `const SessionCookieName = "session"` (recommended) and use it
       for both `Set-Cookie` and `TokenLookup`.
     - Change the login handler to:
       ```go
       var body struct {
           Username string `json:"username"`
           Password string `json:"password"`
       }
       if err := ctx.Bind(&body); err != nil {
           return ctx.JSON(http.StatusBadRequest, ...)
       }
       res, err := h.d.Auth.LoginPlayer(ctx.Request().Context(), body.Username, body.Password)
       ```
     - On success, set the cookie with `Expires: t.Add(constant.SessionLifetime)`
       and `Path: "/"` (keep `Secure`, `HttpOnly`, `SameSite`), then
       `return c.NoContent(http.StatusNoContent)`.
  2. Import `keep-it-up/internal/infrastructure/constant` so `SessionLifetime` is
     available.
  3. Keep error mapping for `usecase.ErrBadRequest` → 400 and
     `usecase.ErrUnauthorized` → 401; keep the generic 500 fallback.
- **Why it needs to be done:**
  TODO 4 makes the OpenAPI spec the contract. A JSON client (per the spec) that
  sends `application/json` would never have its credentials read via
  `FormValue` (which expects `application/x-www-form-urlencoded`), so login would
  always fail from conformant clients. The `session` cookie name is what clients
  will send, so the middleware must look it up there. The 24h cookie vs 72h JWT
  mismatch means an authenticated browser cookie would be invalidated before the
  token — a confusing, non-deterministic session. Finally, the spec's `204` carries
  no content; echoing a JSON body violates HTTP semantics for `204`.

## Architecture & Testing
- **Invariants:**
  - Cookie name is `session` in both the handler and the JWT `TokenLookup`.
  - Cookie expiry equals `constant.SessionLifetime`.
  - Login success returns exactly `204` with an empty body and a `Set-Cookie`
    header named `session`.
  - Spec file `api/openapi.yaml` is unchanged (it already declares `session`).
- **Testing Strategy:**
  Covered by Step 08 handler unit tests: JSON login success (204 + Set-Cookie
  `session`), malformed JSON (400), bad credentials (401).
- **Verification:**
  ```bash
  go build ./...
  ```

## Expected Outcome
- **State of the project:**
  `POST /api/login` accepts a JSON `{username, password}`, sets a `session` cookie
  expiring after `constant.SessionLifetime`, and returns an empty `204`. The JWT
  middleware reads the `session` cookie. The codebase compiles.