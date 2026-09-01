# Backend Plan

Scope: implement the `GET /player/{id}` endpoint (the client's "pretend" endpoint,
now the contract in `api/keep-it-up/openapi.yaml`) and add CORS support so the
Flutter **web** client can reach the API. The **Android emulator** fix is
client-side (base URL `10.0.2.2` + cleartext) and is *not* covered here.

> Reference: `KEEPITUP_README.md`. Assumes the backend architecture described
> there: Go 1.26, Echo v5, SQLite (modernc.org/sqlite) via SQLC, Goose migrations,
> Valkey idempotency, JWT session cookie auth, layered as
> `cmd/` (drivers) · `internal/application` (use cases) · `internal/core` (domain)
> · `internal/infrastructure` (SQLite/SQLC, Valkey, JWT, Clock).

---

## 1. New endpoint: `GET /api/player/{id}`

### 1.1 Contract
- **Path**: `GET /player/{id}` (rooted at `/api`, so full path `/api/player/{id}`)
- **Auth**: session cookie (`session`, a JWT) — required; `401` if missing/invalid
- **Path param**: `id` (`int64`)
- **200** → `{ "id": <int64>, "name": <string>, "username": <string> }`
- **401** → Unauthorized
- **404** → Player not found (or not accessible to the caller)

### 1.2 Client contract this must satisfy
The Flutter client's generated `ApiClient.getPlayerId({ required int id })`
calls this endpoint and deserializes it into the `User` model
(`id`, `name`, `username`). Field names must match exactly (`id`, `name`, `username`).
The client treats `name`/`username` as non-nullable strings.

### 1.3 Server changes by layer

**Domain (`internal/core`)**
- `Player` model already has `id`, `name`, `username` — no change needed.
- If there is a port interface for fetching a player (e.g. a `PlayerQuery`/repository
  port), add `GetPlayerByID(id int64) (Player, error)`; otherwise expose it on the
  existing query surface.

**Infrastructure (`internal/infrastructure`)**
- Add a SQLC query (e.g. in `database/queries/`):
  ```sql
  -- name: GetPlayerByID :one
  SELECT id, name, username FROM players WHERE id = ?;
  ```
- Regenerate: `go tool sqlc generate` → new generated query + `DB → domain` mapping.
- Map the `Player` row to the domain `Player`.

**Application (`internal/application`)**
- Add a use case (e.g. `GetPlayer`) that:
  1. authenticates the caller (session/JWT),
  2. validates `id > 0`,
  3. calls the query,
  4. returns the player, mapping "not found" to a domain error.

**Driver / HTTP (`cmd/server`)**
- Register the route under the `/api` group:
  ```go
  apiGroup.GET("/player/:id", handler.GetPlayer)
  ```
- Handler:
  - parse `:id` as `int64` (bad id → `400`),
  - call the use case,
  - `200` with `{id, name, username}`, or `404` / `401` on the mapped errors.
- Note: Echo path params use `:id`; ensure it does not collide with any existing
  `/player*` route.

### 1.4 Naming note
The backend domain type is `Player`; the client OpenAPI schema for this endpoint is
named `User` (`id`, `name`, `username`). This is only a client-side type name —
wire format is identical. Keep the JSON field names `id`, `name`, `username`.
---

## 2. CORS (required for the Flutter web client)

### 2.1 Problem
The Flutter web app runs on its own origin (e.g. `http://localhost:<random-port>`),
so calls to `http://localhost:8080/api/*` are cross-origin. `POST /login` with
`Content-Type: application/json` triggers an `OPTIONS` preflight; without CORS
headers the browser cancels the request and **no request reaches the server**.

### 2.2 Fix: Echo CORS middleware
Register CORS middleware before the routes (applies to the `/api` group):

```go
e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
    AllowOrigins: allowedOrigins,   // see 2.3
    AllowMethods: []string{
        http.MethodGet,
        http.MethodPost,
        http.MethodOptions,
    },
    AllowHeaders: []string{
        "Content-Type",
        "Idempotency-Key",   // mutations send this header
        "Authorization",
    },
    AllowCredentials: true,  // the session cookie must be sent/stored
    MaxAge: 86400,
}))
```

- **`AllowCredentials: true`** is required because auth uses a cookie.
- Echo's CORS middleware handles the `OPTIONS` preflight (responds `204`) — no
  separate OPTIONS handler needed.

### 2.3 Allowed origins (config)
- Make origins configurable via env, e.g. `CORS_ALLOWED_ORIGINS` (comma-separated).
- Dev default: allow the Flutter web dev origin(s). For local dev you may allow
  `*` only if **not** using credentials; with `AllowCredentials: true`, `*` is not
  valid, so list explicit origins.
- Suggested `.env`:
  ```
  CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080
  ```
  (Add whatever origin `flutter run -d chrome` reports, e.g. `http://localhost:50828`.)
- Production: pin to the real web app origin; never use `*` with credentials.

---

## 3. Validation / tests
- `curl` checks:
  ```bash
  # login to get session cookie
  curl -i -c cookies.txt -X POST http://localhost:8080/api/login \
       -H 'Content-Type: application/json' \
       -d '{"username":"<u>","password":"<p>"}'

  # new endpoint
  curl -i -b cookies.txt http://localhost:8080/api/player/1

  # CORS preflight
  curl -i -X OPTIONS http://localhost:8080/api/login \
       -H 'Origin: http://localhost:50828' \
       -H 'Access-Control-Request-Method: POST' \
       -H 'Access-Control-Request-Headers: content-type,idempotency-key'
  ```
- Go unit tests: use case (`GetPlayer` happy path, not-found, bad id) and the HTTP
  handler (200 / 400 / 401 / 404). Add a CORS middleware test asserting the
  preflight response headers.

---

## 4. Open items / flags
- **`savedBy` semantics**: the client currently renders `Interaction.savedBy` as
  "minutes saved", but the OpenAPI field name suggests it may be a *player id*.
  Confirm and align the backend field meaning before relying on it in the UI.
- **`User` vs `Player` naming**: cosmetic mismatch between client type name and
  backend domain type; wire format is the same.
- **`session` cookie flags**: ensure the cookie is sent on cross-origin (the CORS
  `AllowCredentials` requirement above) and that `SameSite` is set appropriately
  for the web client.
- **Android**: no backend change; the client must use `http://10.0.2.2:8080/api`
  and allow cleartext traffic (client-side).

