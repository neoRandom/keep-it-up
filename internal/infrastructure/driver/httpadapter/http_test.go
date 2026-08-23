package httpadapter

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"keep-it-up/internal/application/usecase"
	"keep-it-up/internal/core/model"
	"keep-it-up/internal/infrastructure/database"
	"keep-it-up/internal/infrastructure/driven"

	"github.com/labstack/echo/v5"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

// --- Fakes implementing the driving ports ---

type fakeTime struct{ now time.Time }

func (f *fakeTime) Time() (time.Time, error) { return f.now, nil }

func newFakeTime() *fakeTime {
	return &fakeTime{now: time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)}
}

// advancingTime is a TimeProvider that returns a base time and advances it by
// one second on each call, so consecutive writes get distinct timestamps that
// satisfy the state-machine trigger's monotonicity constraint.
type advancingTime struct {
	now time.Time
}

func (a *advancingTime) Time() (time.Time, error) {
	t := a.now
	a.now = a.now.Add(time.Second)
	return t, nil
}

func newAdvancingTime() *advancingTime {
	return &advancingTime{now: time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)}
}

type fakeAuth struct {
	loginRes model.AuthResult
	loginErr error
}

func (f *fakeAuth) CheckPlayerPassword(ctx context.Context, username, password string) (database.Player, error) {
	return database.Player{}, nil
}

func (f *fakeAuth) LoginPlayer(ctx context.Context, username, password string) (model.AuthResult, error) {
	return f.loginRes, f.loginErr
}

type fakeFetch struct {
	games        []database.Game
	shared       *model.SharedData
	interactions []database.Interaction
	interaction  *database.Interaction
	err          error
}

func (f *fakeFetch) ListPlayerGames(ctx context.Context, playerId int64) ([]database.Game, error) {
	return f.games, f.err
}

func (f *fakeFetch) GetSharedData(ctx context.Context, gameId int64) (*model.SharedData, error) {
	return f.shared, f.err
}

func (f *fakeFetch) ListInteractions(ctx context.Context, gameId, limit, offset int64) ([]database.Interaction, error) {
	return f.interactions, f.err
}

func (f *fakeFetch) ListPlayerInteractions(ctx context.Context, gameId, playerId, limit, offset int64) ([]database.Interaction, error) {
	return f.interactions, f.err
}

func (f *fakeFetch) FirstInteraction(ctx context.Context, gameId int64) (*database.Interaction, error) {
	return f.interaction, f.err
}

func (f *fakeFetch) LastInteraction(ctx context.Context, gameId int64) (*database.Interaction, error) {
	return f.interaction, f.err
}

type fakeCommands struct {
	saveErr   error
	resumeErr error
	pauseErr  error
}

func (f *fakeCommands) SaveGame(ctx context.Context, gameId, playerId, duration int64) error {
	return f.saveErr
}

func (f *fakeCommands) ResumeGame(ctx context.Context, gameId, playerId int64) error {
	return f.resumeErr
}

func (f *fakeCommands) PauseGame(ctx context.Context, gameId, playerId int64) error {
	return f.pauseErr
}

type fakeAccess struct {
	granted bool
	err     error
}

func (f *fakeAccess) GrantPlayerAccess(ctx context.Context, gameId, playerId int64) error {
	return nil
}

func (f *fakeAccess) RevokePlayerAccess(ctx context.Context, gameId, playerId int64) error {
	return nil
}

func (f *fakeAccess) CheckPlayerAccess(ctx context.Context, gameId, playerId int64) (bool, error) {
	return f.granted, f.err
}

// --- Router construction helpers ---

const testJWTSecret = "test-secret-for-unit-tests"

func newTestRouter(t *testing.T, d Deps) *echo.Echo {
	t.Helper()
	e := echo.New()
	New(":0", testJWTSecret, newFakeTime(), d).routes(e)
	return e
}

// authRequest builds an httptest request carrying a valid JWT for the player in
// the session cookie. The token is produced by the real JwtTokenGenerator so
// the middleware's parsing path (and NewClaimsFunc) is exercised genuinely.
func authRequest(t *testing.T, method, target string, playerID int64, body []byte) *http.Request {
	t.Helper()
	gen := &driven.JwtTokenGenerator{JwtSecret: testJWTSecret, TimeProvider: newFakeTime()}
	token, err := gen.GenerateToken(database.Player{ID: playerID, Username: "neo"})
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, target, bytes.NewReader(body))
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})
	return req
}

func do(t *testing.T, e *echo.Echo, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func decodeBody[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode response body: %v\nbody: %s", err, rec.Body.String())
	}
	return v
}

// --- Login tests ---

func TestLogin(t *testing.T) {
	t.Run("json success returns 204 with session cookie", func(t *testing.T) {
		e := newTestRouter(t, Deps{Auth: &fakeAuth{loginRes: model.AuthResult{Token: "jwt-token"}}})

		req := httptest.NewRequest(http.MethodPost, "/api/login",
			strings.NewReader(`{"username":"neo","password":"secret"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)

		rec := do(t, e, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
		}
		if rec.Body.Len() != 0 {
			t.Errorf("expected empty body for 204, got %q", rec.Body.String())
		}
		setCookie := rec.Header().Get("Set-Cookie")
		if !strings.Contains(setCookie, SessionCookieName) {
			t.Errorf("expected Set-Cookie to contain %q, got %q", SessionCookieName, setCookie)
		}
	})

	t.Run("malformed json returns 400", func(t *testing.T) {
		e := newTestRouter(t, Deps{Auth: &fakeAuth{}})
		req := httptest.NewRequest(http.MethodPost, "/api/login", strings.NewReader("not json"))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := do(t, e, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("bad credentials returns 401", func(t *testing.T) {
		e := newTestRouter(t, Deps{Auth: &fakeAuth{loginErr: usecase.ErrUnauthorized}})
		req := httptest.NewRequest(http.MethodPost, "/api/login",
			strings.NewReader(`{"username":"neo","password":"wrong"}`))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		rec := do(t, e, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
		}
	})
}

// --- Auth guard test (all protected endpoints without cookie → 401) ---

func TestProtectedEndpoints_RequireAuth(t *testing.T) {
	protected := []struct {
		method string
		path   string
		body   []byte
	}{
		{http.MethodGet, "/api/games", nil},
		{http.MethodGet, "/api/shared?gameId=1", nil},
		{http.MethodGet, "/api/interactions?gameId=1", nil},
		{http.MethodPost, "/api/save?gameId=1", []byte(`{"duration":60}`)},
		{http.MethodPost, "/api/play?gameId=1", nil},
		{http.MethodPost, "/api/pause?gameId=1", nil},
	}

	for _, tt := range protected {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			e := newTestRouter(t, Deps{})
			var req *http.Request
			if tt.body != nil {
				req = httptest.NewRequest(tt.method, tt.path, bytes.NewReader(tt.body))
			} else {
				req = httptest.NewRequest(tt.method, tt.path, nil)
			}
			rec := do(t, e, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
			}
		})
	}
}

// --- Read endpoints ---

func TestGetGames(t *testing.T) {
	e := newTestRouter(t, Deps{
		Fetch: &fakeFetch{games: []database.Game{{ID: 1, Name: "Alpha"}, {ID: 2, Name: "Beta"}}},
	})

	rec := do(t, e, authRequest(t, http.MethodGet, "/api/games", 7, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	games := decodeBody[[]gameDTO](t, rec)
	if len(games) != 2 {
		t.Fatalf("len = %d, want 2", len(games))
	}
	if games[0].ID != 1 || games[0].Name != "Alpha" {
		t.Errorf("games[0] = %+v, want {1 Alpha}", games[0])
	}
}

func TestGetShared(t *testing.T) {
	deadline := time.Date(2026, 8, 22, 14, 0, 0, 0, time.UTC)
	valid := true

	t.Run("with access returns shared data", func(t *testing.T) {
		e := newTestRouter(t, Deps{
			Access: &fakeAccess{granted: true},
			Fetch: &fakeFetch{shared: &model.SharedData{
				GameID:     5,
				Status:     model.Playing,
				Valid:      &valid,
				DeadlineAt: &deadline,
			}},
		})
		rec := do(t, e, authRequest(t, http.MethodGet, "/api/shared?gameId=5", 7, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, http.StatusOK, rec.Body.String())
		}
		dto := decodeBody[sharedDataDTO](t, rec)
		if dto.GameID != 5 || dto.Status != "playing" || dto.Valid == nil || !*dto.Valid {
			t.Errorf("unexpected sharedDTO: %+v", dto)
		}
	})

	t.Run("without access returns 404", func(t *testing.T) {
		e := newTestRouter(t, Deps{Access: &fakeAccess{granted: false}})
		rec := do(t, e, authRequest(t, http.MethodGet, "/api/shared?gameId=5", 7, nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("invalid gameId returns 400", func(t *testing.T) {
		e := newTestRouter(t, Deps{})
		rec := do(t, e, authRequest(t, http.MethodGet, "/api/shared?gameId=abc", 7, nil))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})
}

func TestGetInteractions(t *testing.T) {
	interactions := []database.Interaction{
		{ID: 1, GameID: 3, Action: "saved", OccurredAt: "2026-08-22T12:00:00Z"},
	}
	single := &database.Interaction{ID: 1, GameID: 3, Action: "saved", OccurredAt: "2026-08-22T12:00:00Z"}

	t.Run("with access returns interactions (default limit)", func(t *testing.T) {
		e := newTestRouter(t, Deps{
			Access: &fakeAccess{granted: true},
			Fetch:  &fakeFetch{interactions: interactions},
		})
		rec := do(t, e, authRequest(t, http.MethodGet, "/api/interactions?gameId=3", 7, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		dtos := decodeBody[[]interactionDTO](t, rec)
		if len(dtos) != 1 || dtos[0].Action != "saved" {
			t.Errorf("unexpected interactions: %+v", dtos)
		}
	})

	t.Run("without access returns 404", func(t *testing.T) {
		e := newTestRouter(t, Deps{Access: &fakeAccess{granted: false}})
		rec := do(t, e, authRequest(t, http.MethodGet, "/api/interactions?gameId=3", 7, nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("invalid limit returns 400", func(t *testing.T) {
		e := newTestRouter(t, Deps{})
		rec := do(t, e, authRequest(t, http.MethodGet, "/api/interactions?gameId=3&limit=0", 7, nil))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("invalid query value returns 400", func(t *testing.T) {
		e := newTestRouter(t, Deps{
			Access: &fakeAccess{granted: true},
			Fetch:  &fakeFetch{},
		})
		rec := do(t, e, authRequest(t, http.MethodGet, "/api/interactions?gameId=3&query=foo", 7, nil))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("query=all returns a list", func(t *testing.T) {
		e := newTestRouter(t, Deps{
			Access: &fakeAccess{granted: true},
			Fetch:  &fakeFetch{interactions: interactions},
		})
		rec := do(t, e, authRequest(t, http.MethodGet, "/api/interactions?gameId=3&query=all", 7, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		dtos := decodeBody[[]interactionDTO](t, rec)
		if len(dtos) != 1 {
			t.Errorf("unexpected interactions: %+v", dtos)
		}
	})

	t.Run("query=player returns a list", func(t *testing.T) {
		e := newTestRouter(t, Deps{
			Access: &fakeAccess{granted: true},
			Fetch:  &fakeFetch{interactions: interactions},
		})
		rec := do(t, e, authRequest(t, http.MethodGet, "/api/interactions?gameId=3&query=player", 7, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		dtos := decodeBody[[]interactionDTO](t, rec)
		if len(dtos) != 1 {
			t.Errorf("unexpected interactions: %+v", dtos)
		}
	})

	t.Run("query=first returns a single object", func(t *testing.T) {
		e := newTestRouter(t, Deps{
			Access: &fakeAccess{granted: true},
			Fetch:  &fakeFetch{interaction: single},
		})
		rec := do(t, e, authRequest(t, http.MethodGet, "/api/interactions?gameId=3&query=first", 7, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		dto := decodeBody[interactionDTO](t, rec)
		if dto.Action != "saved" {
			t.Errorf("unexpected interaction: %+v", dto)
		}
	})

	t.Run("query=last returns a single object", func(t *testing.T) {
		e := newTestRouter(t, Deps{
			Access: &fakeAccess{granted: true},
			Fetch:  &fakeFetch{interaction: single},
		})
		rec := do(t, e, authRequest(t, http.MethodGet, "/api/interactions?gameId=3&query=last", 7, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		dto := decodeBody[interactionDTO](t, rec)
		if dto.Action != "saved" {
			t.Errorf("unexpected interaction: %+v", dto)
		}
	})

	t.Run("valid offset is accepted", func(t *testing.T) {
		e := newTestRouter(t, Deps{
			Access: &fakeAccess{granted: true},
			Fetch:  &fakeFetch{interactions: interactions},
		})
		rec := do(t, e, authRequest(t, http.MethodGet, "/api/interactions?gameId=3&offset=5", 7, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("invalid offset returns 400", func(t *testing.T) {
		e := newTestRouter(t, Deps{
			Access: &fakeAccess{granted: true},
			Fetch:  &fakeFetch{},
		})
		rec := do(t, e, authRequest(t, http.MethodGet, "/api/interactions?gameId=3&offset=-1", 7, nil))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})
}

// TestGetInteractions_Regression ensures the legacy no-query-param behavior is
// identical to an explicit query=all.
func TestGetInteractions_Regression(t *testing.T) {
	interactions := []database.Interaction{
		{ID: 1, GameID: 3, Action: "saved", OccurredAt: "2026-08-22T12:00:00Z"},
	}

	getBody := func(target string) (string, int) {
		e := newTestRouter(t, Deps{
			Access: &fakeAccess{granted: true},
			Fetch:  &fakeFetch{interactions: interactions},
		})
		rec := do(t, e, authRequest(t, http.MethodGet, target, 7, nil))
		return rec.Body.String(), rec.Code
	}

	noParamBody, noParamStatus := getBody("/api/interactions?gameId=3")
	explicitBody, explicitStatus := getBody("/api/interactions?gameId=3&query=all")

	if noParamStatus != explicitStatus {
		t.Errorf("status mismatch: no-param %d, query=all %d", noParamStatus, explicitStatus)
	}
	if noParamBody != explicitBody {
		t.Errorf("body mismatch:\nno-param:  %s\nquery=all: %s", noParamBody, explicitBody)
	}
}

// --- Command endpoints ---

func TestSaveGame(t *testing.T) {
	t.Run("success returns 204", func(t *testing.T) {
		e := newTestRouter(t, Deps{
			Access:   &fakeAccess{granted: true},
			Commands: &fakeCommands{},
		})
		rec := do(t, e, authRequest(t, http.MethodPost, "/api/save?gameId=3", 7, []byte(`{"duration":60}`)))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, http.StatusNoContent, rec.Body.String())
		}
	})

	t.Run("without access returns 404", func(t *testing.T) {
		e := newTestRouter(t, Deps{Access: &fakeAccess{granted: false}})
		rec := do(t, e, authRequest(t, http.MethodPost, "/api/save?gameId=3", 7, []byte(`{"duration":60}`)))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("invalid duration returns 400", func(t *testing.T) {
		e := newTestRouter(t, Deps{Access: &fakeAccess{granted: true}})
		rec := do(t, e, authRequest(t, http.MethodPost, "/api/save?gameId=3", 7, []byte(`{"duration":0}`)))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("state machine conflict returns 409", func(t *testing.T) {
		e := newTestRouter(t, Deps{
			Access:   &fakeAccess{granted: true},
			Commands: &fakeCommands{saveErr: errors.New("cannot save while paused")},
		})
		rec := do(t, e, authRequest(t, http.MethodPost, "/api/save?gameId=3", 7, []byte(`{"duration":60}`)))
		if rec.Code != http.StatusConflict {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusConflict)
		}
	})
}

// --- Integration test (real use cases against in-memory SQLite) ---

// newIntegrationRouter builds a router wired to real use cases backed by an
// in-memory SQLite database with the Goose migrations applied.
type integrationDeps struct {
	e        *echo.Echo
	commands *usecase.GameCommands
	access   *usecase.AccessManagement
}

func newIntegrationRouter(t *testing.T) *integrationDeps {
	t.Helper()
	goose.SetLogger(goose.NopLogger())

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := goose.SetDialect("sqlite3"); err != nil {
		t.Fatalf("set goose dialect: %v", err)
	}
	dir, ok := findMigrations(t)
	if !ok {
		t.Fatal("could not find migrations dir")
	}
	if err := goose.Up(db, dir); err != nil {
		t.Fatalf("run goose migrations: %v", err)
	}

	q := database.New(db)
	fetch := usecase.NewDataFetching(q, newFakeTime())
	// Use an advancing clock so consecutive commands persist distinct,
	// monotonically increasing timestamps (required by the state-machine trigger).
	commands := usecase.NewGameCommands(q, newAdvancingTime())
	access := usecase.NewAccessManagement(q)

	e := echo.New()
	New(":0", testJWTSecret, newFakeTime(), Deps{
		Fetch:  fetch,
		Access: access,
	}).routes(e)
	return &integrationDeps{e: e, commands: commands, access: access}
}

func findMigrations(t *testing.T) (string, bool) {
	t.Helper()
	// Test CWD is the package dir (internal/infrastructure/driver/httpadapter),
	// five levels below the repo root. Try the repo-relative path and climbing
	// up enough levels to reach database/migrations from it.
	cwd, err := os.Getwd()
	if err == nil {
		if info, statErr := os.Stat(filepath.Join(cwd, "database", "migrations")); statErr == nil && info.IsDir() {
			return filepath.Join(cwd, "database", "migrations"), true
		}
	}
	for _, p := range []string{
		"database/migrations",
		filepath.Join("..", "..", "..", "database", "migrations"),
		filepath.Join("..", "..", "..", "..", "database", "migrations"),
		filepath.Join(cwd, "..", "..", "..", "..", "database", "migrations"),
	} {
		if info, statErr := os.Stat(p); statErr == nil && info.IsDir() {
			return p, true
		}
	}
	return "", false
}

func TestGetInteractions_Integration(t *testing.T) {
	ctx := context.Background()

	t.Run("query=player returns the authenticated player's interactions", func(t *testing.T) {
		d := newIntegrationRouter(t)
		if err := d.access.GrantPlayerAccess(ctx, 1, 1); err != nil {
			t.Fatalf("setup GrantPlayerAccess: %v", err)
		}
		// Player 1 saves in game 1 (12:00:00) and again (12:00:01); the fixed
		// clock advances one second per call through the real commands use case.
		if err := d.commands.SaveGame(ctx, 1, 1, 60); err != nil {
			t.Fatalf("setup SaveGame 1: %v", err)
		}
		if err := d.commands.SaveGame(ctx, 1, 1, 60); err != nil {
			t.Fatalf("setup SaveGame 2: %v", err)
		}

		rec := do(t, d.e, authRequest(t, http.MethodGet, "/api/interactions?gameId=1&query=player", 1, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, http.StatusOK, rec.Body.String())
		}
		dtos := decodeBody[[]interactionDTO](t, rec)
		if len(dtos) != 2 {
			t.Fatalf("len = %d, want 2", len(dtos))
		}
		if dtos[0].Action != "saved" || dtos[1].Action != "saved" {
			t.Errorf("unexpected actions: %+v", dtos)
		}
	})

	t.Run("query=first returns the earliest interaction", func(t *testing.T) {
		d := newIntegrationRouter(t)
		if err := d.access.GrantPlayerAccess(ctx, 1, 1); err != nil {
			t.Fatalf("setup GrantPlayerAccess: %v", err)
		}
		if err := d.commands.SaveGame(ctx, 1, 1, 60); err != nil {
			t.Fatalf("setup SaveGame 1: %v", err)
		}
		if err := d.commands.SaveGame(ctx, 1, 1, 60); err != nil {
			t.Fatalf("setup SaveGame 2: %v", err)
		}

		rec := do(t, d.e, authRequest(t, http.MethodGet, "/api/interactions?gameId=1&query=first", 1, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, http.StatusOK, rec.Body.String())
		}
		dto := decodeBody[interactionDTO](t, rec)
		if dto.OccurredAt != "2026-08-22T12:00:00Z" {
			t.Errorf("first occurredAt = %q, want the initial save", dto.OccurredAt)
		}
	})

	t.Run("query=last returns the latest interaction", func(t *testing.T) {
		d := newIntegrationRouter(t)
		if err := d.access.GrantPlayerAccess(ctx, 1, 1); err != nil {
			t.Fatalf("setup GrantPlayerAccess: %v", err)
		}
		if err := d.commands.SaveGame(ctx, 1, 1, 60); err != nil {
			t.Fatalf("setup SaveGame 1: %v", err)
		}
		if err := d.commands.SaveGame(ctx, 1, 1, 60); err != nil {
			t.Fatalf("setup SaveGame 2: %v", err)
		}

		rec := do(t, d.e, authRequest(t, http.MethodGet, "/api/interactions?gameId=1&query=last", 1, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, http.StatusOK, rec.Body.String())
		}
		dto := decodeBody[interactionDTO](t, rec)
		if dto.OccurredAt != "2026-08-22T12:00:01Z" {
			t.Errorf("last occurredAt = %q, want the final save", dto.OccurredAt)
		}
	})
}

func TestPlayPause(t *testing.T) {
	t.Run("play success returns 204", func(t *testing.T) {
		e := newTestRouter(t, Deps{Access: &fakeAccess{granted: true}, Commands: &fakeCommands{}})
		rec := do(t, e, authRequest(t, http.MethodPost, "/api/play?gameId=3", 7, nil))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("play status = %d, want %d", rec.Code, http.StatusNoContent)
		}
	})

	t.Run("play without access returns 404", func(t *testing.T) {
		e := newTestRouter(t, Deps{Access: &fakeAccess{granted: false}})
		rec := do(t, e, authRequest(t, http.MethodPost, "/api/play?gameId=3", 7, nil))
		if rec.Code != http.StatusNotFound {
			t.Fatalf("play status = %d, want %d", rec.Code, http.StatusNotFound)
		}
	})

	t.Run("play conflict returns 409", func(t *testing.T) {
		e := newTestRouter(t, Deps{
			Access:   &fakeAccess{granted: true},
			Commands: &fakeCommands{resumeErr: errors.New("cannot resume")},
		})
		rec := do(t, e, authRequest(t, http.MethodPost, "/api/play?gameId=3", 7, nil))
		if rec.Code != http.StatusConflict {
			t.Fatalf("play status = %d, want %d", rec.Code, http.StatusConflict)
		}
	})

	t.Run("pause success returns 204", func(t *testing.T) {
		e := newTestRouter(t, Deps{Access: &fakeAccess{granted: true}, Commands: &fakeCommands{}})
		rec := do(t, e, authRequest(t, http.MethodPost, "/api/pause?gameId=3", 7, nil))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("pause status = %d, want %d", rec.Code, http.StatusNoContent)
		}
	})

	t.Run("pause conflict returns 409", func(t *testing.T) {
		e := newTestRouter(t, Deps{
			Access:   &fakeAccess{granted: true},
			Commands: &fakeCommands{pauseErr: errors.New("cannot pause")},
		})
		rec := do(t, e, authRequest(t, http.MethodPost, "/api/pause?gameId=3", 7, nil))
		if rec.Code != http.StatusConflict {
			t.Fatalf("pause status = %d, want %d", rec.Code, http.StatusConflict)
		}
	})
}