package httpadapter

import (
	"context"
	"net/http"
	"strconv"
	"testing"
	"time"

	"keep-it-up/internal/core/model"
	"keep-it-up/internal/infrastructure/constant"

	"github.com/labstack/echo/v5"
)

// --- Fakes implementing the idempotency port (IdempotencyStore) ---

type idemRecord struct {
	status string
	code   int
	body   []byte
}

// fakeIdempotencyStore is an in-memory IdempotencyStore that mirrors the real
// state machine: Acquire creates IN_PROGRESS once, Complete flips it to
// COMPLETED. Tests may pre-populate keys to simulate an in-flight or completed
// record.
type fakeIdempotencyStore struct {
	keys          map[string]idemRecord
	completeCalls int
}

func newFakeIdempotencyStore() *fakeIdempotencyStore {
	return &fakeIdempotencyStore{keys: map[string]idemRecord{}}
}

func (f *fakeIdempotencyStore) Acquire(_ context.Context, key string, _ time.Duration) (bool, error) {
	if _, ok := f.keys[key]; ok {
		return false, nil
	}
	f.keys[key] = idemRecord{status: constant.IdempotencyStatusInProgress}
	return true, nil
}

func (f *fakeIdempotencyStore) State(_ context.Context, key string) (string, int, []byte, error) {
	rec, ok := f.keys[key]
	if !ok {
		return "", 0, nil, nil
	}
	return rec.status, rec.code, rec.body, nil
}

func (f *fakeIdempotencyStore) Complete(_ context.Context, key string, code int, body []byte, _ time.Duration) error {
	f.completeCalls++
	rec := f.keys[key]
	rec.status = constant.IdempotencyStatusCompleted
	rec.code = code
	rec.body = body
	f.keys[key] = rec
	return nil
}

// countingCommands records how many times the SaveGame use case is invoked so
// tests can prove at-most-once execution.
type countingCommands struct {
	saveCalls int
}

func (c *countingCommands) SaveGame(_ context.Context, _, _ int64, _ int64) error {
	c.saveCalls++
	return nil
}

func (c *countingCommands) ResumeGame(_ context.Context, _, _ int64) error { return nil }

func (c *countingCommands) PauseGame(_ context.Context, _, _ int64) error { return nil }

// countingAuth records login invocations for the /login replay invariant.
type countingAuth struct {
	loginCalls int
	loginRes   model.AuthResult
}

func (a *countingAuth) LoginPlayer(_ context.Context, _, _ string) (model.AuthResult, error) {
	a.loginCalls++
	return a.loginRes, nil
}

func (a *countingAuth) CheckPlayerPassword(_ context.Context, _, _ string) (model.Player, error) {
	return model.Player{}, nil
}

// --- Router helper ---

// newIdempotentRouter builds a router with idempotency enabled against the
// given in-memory store, using a fixed header name for determinism.
func newIdempotentRouter(t *testing.T, d Deps, store IdempotencyStore) *echo.Echo {
	t.Helper()
	e := echo.New()
	New(":0", testJWTSecret, newFakeTime(), d, 72*time.Hour, 20, WithIdempotency(store, time.Minute, "Idempotency-Key")).routes(e)
	return e
}

// idemRequest is authRequest plus an optional Idempotency-Key header.
func idemRequest(t *testing.T, method, target string, playerID int64, body []byte, key string) *http.Request {
	t.Helper()
	req := authRequest(t, method, target, playerID, body)
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	return req
}

// storeKeyForUser mirrors the middleware's scoped key format for pre-seeding.
func storeKeyForUser(playerID int64, rawKey string) string {
	return "idempotency:user:" + strconv.FormatInt(playerID, 10) + ":" + rawKey
}

func TestIdempotency_NoHeaderPassesThrough(t *testing.T) {
	cmds := &countingCommands{}
	store := newFakeIdempotencyStore()
	e := newIdempotentRouter(t, Deps{Access: &fakeAccess{granted: true}, Commands: cmds}, store)

	rec := do(t, e, authRequest(t, http.MethodPost, "/api/save?gameId=1", 7, []byte(`{"duration":60}`)))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if cmds.saveCalls != 1 {
		t.Fatalf("saveCalls = %d, want 1", cmds.saveCalls)
	}
	if len(store.keys) != 0 {
		t.Fatalf("store should be untouched without a key, got %d entries", len(store.keys))
	}
}

func TestIdempotency_InvalidKeyReturnsBadRequest(t *testing.T) {
	cmds := &countingCommands{}
	e := newIdempotentRouter(t, Deps{Access: &fakeAccess{granted: true}, Commands: cmds}, newFakeIdempotencyStore())

	req := authRequest(t, http.MethodPost, "/api/save?gameId=1", 7, []byte(`{"duration":60}`))
	req.Header.Set("Idempotency-Key", "bad key with spaces!")
	rec := do(t, e, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if cmds.saveCalls != 0 {
		t.Fatalf("saveCalls = %d, want 0 (invalid key must not run use case)", cmds.saveCalls)
	}
}

func TestIdempotency_NewKeyCompletesAndCaches(t *testing.T) {
	cmds := &countingCommands{}
	store := newFakeIdempotencyStore()
	e := newIdempotentRouter(t, Deps{Access: &fakeAccess{granted: true}, Commands: cmds}, store)

	rec := do(t, e, idemRequest(t, http.MethodPost, "/api/save?gameId=1", 7, []byte(`{"duration":60}`), "key-1"))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if cmds.saveCalls != 1 {
		t.Fatalf("saveCalls = %d, want 1", cmds.saveCalls)
	}
	if store.completeCalls != 1 {
		t.Fatalf("completeCalls = %d, want 1", store.completeCalls)
	}

	key := storeKeyForUser(7, "key-1")
	rec2, ok := store.keys[key]
	if !ok {
		t.Fatalf("store missing key %q", key)
	}
	if rec2.status != constant.IdempotencyStatusCompleted {
		t.Fatalf("status = %q, want %q", rec2.status, constant.IdempotencyStatusCompleted)
	}
	if rec2.code != http.StatusNoContent {
		t.Fatalf("cached code = %d, want %d", rec2.code, http.StatusNoContent)
	}
	if len(rec2.body) != 0 {
		t.Fatalf("cached body = %q, want empty", rec2.body)
	}
}

func TestIdempotency_InProgressReturnsConflict(t *testing.T) {
	cmds := &countingCommands{}
	store := newFakeIdempotencyStore()
	store.keys[storeKeyForUser(7, "dup")] = idemRecord{status: constant.IdempotencyStatusInProgress}
	e := newIdempotentRouter(t, Deps{Access: &fakeAccess{granted: true}, Commands: cmds}, store)

	rec := do(t, e, idemRequest(t, http.MethodPost, "/api/save?gameId=1", 7, []byte(`{"duration":60}`), "dup"))
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusConflict)
	}
	if cmds.saveCalls != 0 {
		t.Fatalf("saveCalls = %d, want 0 (in-progress must not run use case)", cmds.saveCalls)
	}
}

func TestIdempotency_CompletedReplaysBody(t *testing.T) {
	cmds := &countingCommands{}
	store := newFakeIdempotencyStore()
	cached := []byte(`{"message":"Game is not currently playing"}`)
	store.keys[storeKeyForUser(7, "cached")] = idemRecord{
		status: constant.IdempotencyStatusCompleted,
		code:   http.StatusConflict,
		body:   cached,
	}
	e := newIdempotentRouter(t, Deps{Access: &fakeAccess{granted: true}, Commands: cmds}, store)

	rec := do(t, e, idemRequest(t, http.MethodPost, "/api/save?gameId=1", 7, []byte(`{"duration":60}`), "cached"))
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusConflict)
	}
	if rec.Body.String() != string(cached) {
		t.Fatalf("body = %q, want %q", rec.Body.String(), cached)
	}
	if cmds.saveCalls != 0 {
		t.Fatalf("saveCalls = %d, want 0 (completed must replay without re-running use case)", cmds.saveCalls)
	}
}

func TestIdempotency_CompletedReplaysNoContent(t *testing.T) {
	cmds := &countingCommands{}
	store := newFakeIdempotencyStore()
	store.keys[storeKeyForUser(7, "done")] = idemRecord{
		status: constant.IdempotencyStatusCompleted,
		code:   http.StatusNoContent,
	}
	e := newIdempotentRouter(t, Deps{Access: &fakeAccess{granted: true}, Commands: cmds}, store)

	rec := do(t, e, idemRequest(t, http.MethodPost, "/api/save?gameId=1", 7, []byte(`{"duration":60}`), "done"))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", rec.Body.String())
	}
	if cmds.saveCalls != 0 {
		t.Fatalf("saveCalls = %d, want 0", cmds.saveCalls)
	}
}

// TestIdempotency_Invariant_RepeatedRequestRunsOnce is the core guarantee: two
// identical mutation requests sharing a key execute the use case exactly once.
func TestIdempotency_Invariant_RepeatedRequestRunsOnce(t *testing.T) {
	cmds := &countingCommands{}
	store := newFakeIdempotencyStore()
	e := newIdempotentRouter(t, Deps{Access: &fakeAccess{granted: true}, Commands: cmds}, store)

	body := []byte(`{"duration":60}`)
	first := do(t, e, idemRequest(t, http.MethodPost, "/api/save?gameId=1", 7, body, "idem-key"))
	if first.Code != http.StatusNoContent {
		t.Fatalf("first status = %d, want %d", first.Code, http.StatusNoContent)
	}

	second := do(t, e, idemRequest(t, http.MethodPost, "/api/save?gameId=1", 7, body, "idem-key"))
	if second.Code != http.StatusNoContent {
		t.Fatalf("second status = %d, want %d", second.Code, http.StatusNoContent)
	}

	if cmds.saveCalls != 1 {
		t.Fatalf("saveCalls = %d, want exactly 1 (use case must not run twice)", cmds.saveCalls)
	}
}

func TestIdempotency_LoginReplay(t *testing.T) {
	auth := &countingAuth{loginRes: model.AuthResult{Token: "jwt-token"}}
	store := newFakeIdempotencyStore()
	e := newIdempotentRouter(t, Deps{Auth: auth}, store)

	build := func() *http.Request {
		req := authRequest(t, http.MethodPost, "/api/login", 0, []byte(`{"username":"neo","password":"secret"}`))
		req.Header.Set("Idempotency-Key", "login-key")
		return req
	}

	first := do(t, e, build())
	if first.Code != http.StatusNoContent {
		t.Fatalf("first status = %d, want %d", first.Code, http.StatusNoContent)
	}
	if auth.loginCalls != 1 {
		t.Fatalf("loginCalls after first = %d, want 1", auth.loginCalls)
	}

	second := do(t, e, build())
	if second.Code != http.StatusNoContent {
		t.Fatalf("second status = %d, want %d", second.Code, http.StatusNoContent)
	}
	if auth.loginCalls != 1 {
		t.Fatalf("loginCalls after replay = %d, want 1 (login must not re-execute)", auth.loginCalls)
	}
}
