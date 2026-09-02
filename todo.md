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

---

# Backend TODO — Save-by-calculus fix (issue #1)

This is a **backend** change. The client already depends on the server producing a
correct `deadlineAt`; it is documented here so the backend can be fixed without
touching the client.

## Problem

When a player saves *after* the current deadline has already passed, the new
deadline is computed by extending the **old (missed)** deadline instead of
anchoring at the moment of the save. As a result the game can remain overdue even
after a successful save.

Example: the deadline was at `T`. At `T + 5min` a player saves for `4min`.
Expected result: `deadline = T + 5min + 4min = T + 9min` (valid again).
Current result: `deadline = T + 4min`, which is still `1min` in the past — the game
is still overdue.

## Location

- Repo: `keep-it-up` (the Go backend)
- File: `internal/core/service/shared_data.go`
- Function: `BuildSharedData`
- Branch: the `"saved"` case, specifically the `else` arm for subsequent saves:

```go
} else {
    deadline := data.DeadlineAt.Add(extension) // BUG: anchored to previous deadline
    data.DeadlineAt = &deadline
}
```

The first save (`status == NotStarted`) is already correct because it anchors at the
interaction's own time:

```go
if data.Status == model.NotStarted {
    deadline := ia.OccurredAt.Add(extension)
    data.DeadlineAt = &deadline
    data.Status = model.Playing
}
```

## Fix

Make subsequent saves anchor at the save's own time too, so the new deadline always
equals `save time + duration`, regardless of whether a previous deadline was missed:

```go
} else {
    // Anchor at the save's own time so a missed previous deadline is "paid back"
    // (the new deadline is now + extension, never in the past).
    deadline := ia.OccurredAt.Add(extension)
    data.DeadlineAt = &deadline
}
```

`ia.OccurredAt` is the server-side timestamp of the save interaction (set by the
`GameCommands.SaveGame` use case), i.e. the current time at which the save happened.
This "includes the time since the last deadline" by resetting the anchor to `now`.

## Tests

Add a case to `internal/core/service/shared_data_test.go` that reproduces the bug:

```go
func TestBuildSharedData_SaveAfterDeadlineIsNotOverdue(t *testing.T) {
    // Save 1 at 12:00:00 with 60s => deadline 12:01:00.
    // Save 2 occurs AFTER that deadline at 12:06:00 with 240s.
    // The new deadline must be 12:06:00 + 240s = 12:10:00 (never in the past).
    interactions := []model.Interaction{
        {
            ID:         1,
            GameID:     5,
            PlayerID:   intPtr(1),
            Action:     "saved",
            OccurredAt: time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC),
            SavedBy:    intPtr(60),
        },
        {
            ID:         2,
            GameID:     5,
            PlayerID:   intPtr(1),
            Action:     "saved",
            OccurredAt: time.Date(2026, 8, 22, 12, 6, 0, 0, time.UTC),
            SavedBy:    intPtr(240),
        },
    }

    shared, err := BuildSharedData(5, interactions, time.Date(2026, 8, 22, 12, 6, 0, 0, time.UTC))
    if err != nil {
        t.Fatalf("BuildSharedData: %v", err)
    }

    want := time.Date(2026, 8, 22, 12, 10, 0, 0, time.UTC)
    if shared.DeadlineAt == nil || !shared.DeadlineAt.Equal(want) {
        t.Errorf("DeadlineAt = %v, want %v", shared.DeadlineAt, want)
    }
    if shared.Valid == nil || !*shared.Valid {
        t.Errorf("Valid = %v, want true (game must be valid again after the save)", shared.Valid)
    }
}
```

Run it with `go test ./internal/core/service/...`.
