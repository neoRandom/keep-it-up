# Code Review — keep-it-up (final)

Two Critical findings, three Moderate, two Minor. The Critical findings are related: a shared misunderstanding of SQLC's "no rows" contract that produced a real auth vulnerability, and — separately — a test that encodes the vulnerability as expected behavior.

---

## CRITICAL

### C1. Unknown username is not treated as authentication failure — username enumeration + false 500s

**`internal/application/usecase/authentication.go`, `CheckPlayerPassword`:**

```go
player, err := uc.q.GetPlayerByUsername(ctx, username)
if err != nil {
    return model.Player{}, fmt.Errorf("failed to get player by username %q: %w", username, err)
}
```

`GetPlayerByUsername` is a SQLC `:one` query — on no match it returns `sql.ErrNoRows`, not a zero-value struct. That error is wrapped and returned as a **generic error**, which is a fundamentally different signal than the wrong-password path two lines later, which returns `model.Player{}, nil` (zero-ID player, **no error**).

`LoginPlayer` compounds this:

```go
player, err := uc.CheckPlayerPassword(ctx, username, password)
if err != nil {
    return model.AuthResult{}, fmt.Errorf("failed to check if password is correct: %w", err)
}
if player.ID == 0 {
    return model.AuthResult{}, ErrUnauthorized
}
```

Wrong password → `player.ID == 0`, `err == nil` → `ErrUnauthorized` → HTTP 401 (correct).
Unknown username → `err != nil` (wrapping `sql.ErrNoRows`) → **not** `ErrUnauthorized` → falls through `handleLogin`'s switch to the generic `case err != nil` → **HTTP 500**, and the raw username is written to the server log via `log.Printf("login error: %v", err)`.

**Impact:** an unauthenticated caller can distinguish "wrong password" from "no such account" purely from HTTP status code (401 vs 500) — textbook username enumeration — and a credential-stuffing scan against a username list will flood error logs/500-rate metrics with noise, degrading real incident detection. This is a security defect, not a style issue.

**Root cause confirmed elsewhere too:** `player_management.go`'s `UpdatePlayerPasswordForce` has the identical misunderstanding — `if player.ID == 0 { return errors.New("player does not exist") }` is dead code, because a `sql.ErrNoRows` from the preceding `GetPlayerByUsername` call already returns on the line above it. Same bug pattern, no live consequence there since the outer message isn't security-sensitive, but it confirms this isn't a one-off typo — it's a systemic assumption about SQLC's not-found contract.

**Fix:**
```go
player, err := uc.q.GetPlayerByUsername(ctx, username)
if err != nil {
    if errors.Is(err, sql.ErrNoRows) {
        return model.Player{}, nil // indistinguishable from wrong password
    }
    return model.Player{}, fmt.Errorf("failed to get player by username: %w", err)
}
```
Apply the same fix in `UpdatePlayerPasswordForce`, and remove the now-genuinely-reachable `player.ID == 0` check there (or keep it as defense-in-depth, now that it can actually trigger).

### C2. Test asserts the bug as correct behavior (Test Validity)

`internal/application/usecase/authentication_test.go`:

```go
func TestAuthentication_CheckPlayerPasswordRejectsUnknownUser(t *testing.T) {
    ...
    player, err := auth.CheckPlayerPassword(ctx, "ghost", "secret123")
    if err == nil {
        t.Fatal("CheckPlayerPassword() did not return an error for an unknown username")
    }
    ...
}
```

This doesn't test the invariant ("an unknown username must be rejected the same way a wrong password is") — it tests the *current implementation's accidental behavior* (returns an error at all). It's the same defect class the original audit flagged in this exact file (tests passing for the wrong reason) — Phase A fixed one instance and this instance was never caught, because nothing in the audit specifically re-scanned for "does this test assert the correct outcome, or just *an* outcome."

Nothing at the HTTP layer closes this gap either: `TestLogin`'s unauthorized case injects `usecase.ErrUnauthorized` directly into a hand-rolled fake (`fakeAuth{loginErr: usecase.ErrUnauthorized}`) — it proves the handler maps that specific sentinel to 401, but never exercises the real `LoginPlayer`→`CheckPlayerPassword` path with an actual unknown username, so the integration seam where C1 lives is untested end-to-end in either direction.

**Fix:** after C1's fix, this test should assert `err == nil && player.ID == 0` — identical shape to the wrong-password test, ideally merged into one table-driven test with `{"wrong password", "wrongpass"}` and `{"unknown user", n/a}` as two rows proving the same outcome. Add one HTTP-layer test that exercises the real usecase (not the fake) with a genuinely unknown username end-to-end, asserting 401 — this is the class of test the fakes cannot catch by construction.

### C3. Unbounded `duration` → silent `int64` overflow corrupts game deadline

`GameCommands.SaveGame` validates only `duration < 1`:
```go
if duration < 1 {
    return errors.New("save duration cannot be less than 1 second")
}
```
No upper bound, anywhere — not in the use case, not in the HTTP handler (`if body.Duration < 1`), not in the DB (`CHECK (saved_by IS NULL OR saved_by > 0)` — positive, unbounded).

That value flows into `shared_data.go`:
```go
extension := time.Duration(*ia.SavedBy) * time.Second
```
`time.Duration` is `int64` nanoseconds. Go integer multiplication overflow is **silent** (wraps, no panic). `math.MaxInt64 / int64(time.Second) ≈ 9.22e9` seconds (~292 years). A client submitting `duration` above that — a plausible int64 value, not an edge-of-range attack — silently wraps to an incorrect (possibly negative) `time.Duration`, producing a corrupted `DeadlineAt` that `ComputeValid` then evaluates against. This is silent state corruption from untrusted input reaching core business logic, not a crash — worse, because it's undetectable without inspecting the resulting deadline.

No test exercises this: `game_commands_test.go`'s only duration boundary test is `{0, -5}` (lower bound only).

**Fix:** cap `duration` at the use-case boundary (e.g., a `constant.MaxSaveDurationSeconds`, chosen well under the overflow threshold — game sessions don't need centuries of extension) and reject above it in `SaveGame`, mirroring the existing lower-bound check. Add a test asserting rejection at/above the cap and a `BuildSharedData` test with a duration near `math.MaxInt64/int64(time.Second)` proving no silent wrap.

---

## MODERATE

### M1. Brittle DB-trigger string-matching survives the Phase C boundary fix — just moved, not removed

The original audit (finding 2a) flagged `handlers.go`'s `conflictStatusFromErr` string-matching SQLite trigger text. That's gone from `handlers.go` — `commandError` now correctly uses `errors.Is` against sentinel errors. But the string-matching itself only **relocated** to `game_commands.go`:

```go
if strings.Contains(err.Error(), "cannot save while paused") {
    return ErrCannotSaveWhilePaused
}
```
against a trigger that raises `RAISE(ABORT, 'cannot save while paused')`, and similarly for `"cannot pause"` / `"cannot resume"` against `'cannot pause: game is not currently playing'` / `'cannot resume: game is not currently paused'`.

This is now a single-layer translation (better than being re-matched at the driver too) but the fragility is unchanged: any future edit to the trigger's message text — even a typo fix in the migration file — silently stops the sentinel from firing, and every state-machine conflict becomes an unclassified 500 instead of a 409, with no compiler error and no test failure unless someone specifically re-runs the string-match test with the exact updated wording. The existing tests (`game_commands_test.go`) run against a real DB and currently pass because the strings genuinely match today — they don't protect against drift.

**Fix:** give the trigger distinct, greppable error codes instead of prose (e.g. `RAISE(ABORT, 'ERR_CANNOT_SAVE_WHILE_PAUSED')`), or — better — move the state check out of the DB trigger and into `BuildSharedData`/the use case itself (the domain already has `ComputeValid`/state-transition logic in `shared_data.go`; the trigger duplicates a decision the domain can and should make before issuing the write), making the sentinel-error path independent of SQL string content entirely.

### M2. `database/sql` types constructed directly in the application layer

`game_commands.go` and `data_fetching.go` (`ListPlayerInteractions`) build `sql.NullInt64{Int64: playerId, Valid: true}` inline to populate `*Params` structs. The "no repository layer" decision justifies use cases calling `*database.Queries` directly, but constructing a `database/sql` wrapper type by hand inside business logic is a step further — it's the application layer authoring persistence-null-encoding, not just invoking persistence. It doesn't rise to the severity of C1–C3, but it's inconsistent with the effort just spent making `internal/core` free of infrastructure knowledge — the leak just sits one layer up, at the use case, where the audit didn't look.

**Fix:** a tiny local helper (`func nullInt64(v int64) sql.NullInt64`) inside the usecase package, or accept this as the accepted cost of "use cases talk to SQLC directly" and document it as such — either is defensible, but it should be a decision, not an oversight.

### M3. `IsAlphanumeric` (usernames) and `IsPasswordValid` (passwords) enforce different alphabets with no stated reason

`util.IsAlphanumeric` (usernames) uses `unicode.IsLetter`/`unicode.IsDigit` — accepts any Unicode letter. `service.IsPasswordValid` restricts to strict ASCII `[A-Za-z0-9]`. A username can contain characters a password cannot. This may be intentional (ASCII-only passwords sidestep normalization/homoglyph issues at hash time), but nothing documents it, and it's the kind of asymmetry a future maintainer will "fix" into consistency without realizing one side was deliberate.

**Fix:** one-line doc comment on `IsPasswordValid` stating the ASCII-only rule is deliberate (and why), or align the two if the difference is accidental.

---

## MINOR

### N1. No upper bound on `limit` for interaction listing
`interactionsLimit` rejects `< 1` but nothing caps the top end — a client can request `limit=100000000` and the use case passes it straight to `ListRecentInteractions`. Low severity (SQLite will just do a large scan, not crash), but free to fix: cap at the HTTP boundary alongside the existing default.

### N2. `ToDomainInteractions`' fail-fast doc comment is accurate but the failure is currently untriggerable in tests
Confirmed correct behavior (nil slice, first-error-wins) from the prior review round — no action needed, noted only because it's the one place in the mapping layer with a fallible contract and zero test exercising the failure path itself (only the success path is tested via `data_fetching_test.go`). Add one test constructing a `database.Interaction` with a malformed `OccurredAt` string to lock in the fail-fast/nil-slice contract the doc comment promises.
