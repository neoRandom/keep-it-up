package httpadapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"keep-it-up/internal/application/model"
	"keep-it-up/internal/application/usecase"
	"keep-it-up/internal/core/port"
	"keep-it-up/internal/infrastructure/constant"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v5"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

// SessionCookieName matches the `sessionCookie` security scheme in api/openapi.yaml.
const SessionCookieName string = "session"

// defaultInteractionsLimit is the LIMIT used when a client omits the
// `limit` query param (spec default).
const defaultInteractionsLimit int64 = 20

// errStop signals that a handler has already written a response and must not
// proceed (e.g. because access was denied).
var errStop = errors.New("stop handler")

type Deps struct {
	Auth     port.Authentication
	Fetch    port.DataFetching
	Commands port.GameCommands
	Access   port.AccessManagement
}

type gameDTO struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type sharedDataDTO struct {
	GameID       int64      `json:"gameId"`
	Status       string     `json:"status"`
	Valid        *bool      `json:"valid"`
	DeadlineAt   *time.Time `json:"deadlineAt"`
	LastSavedAt  *time.Time `json:"lastSavedAt"`
	LastPausedAt *time.Time `json:"lastPausedAt"`
}

type interactionDTO struct {
	ID         int64  `json:"id"`
	GameID     int64  `json:"gameId"`
	PlayerID   *int64 `json:"playerId"`
	Action     string `json:"action"`
	OccurredAt string `json:"occurredAt"`
	SavedBy    *int64 `json:"savedBy"`
}

type saveRequest struct {
	Duration int64 `json:"duration"`
}

type HTTPAdapter struct {
	addr      string
	jwtSecret string
	tp        port.TimeProvider
	d         Deps
}

func New(addr string, jwtSecret string, tp port.TimeProvider, d Deps) *HTTPAdapter {
	return &HTTPAdapter{
		addr:      addr,
		jwtSecret: jwtSecret,
		tp:        tp,
		d:         d,
	}
}

// routes registers every HTTP route and middleware. It is separate from Run so
// handlers can be unit-tested with Echo's test helpers.
func (h *HTTPAdapter) routes(e *echo.Echo) {
	unprotected := e.Group("/api")
	unprotected.POST("/login", h.handleLogin)

	api := unprotected.Group("")
	api.Use(echojwt.WithConfig(echojwt.Config{
		SigningKey:  []byte(h.jwtSecret),
		TokenLookup: fmt.Sprintf("cookie:%s", SessionCookieName),
		// Use typed claims so handlers can read the actor's UserID. Without
		// this the middleware defaults to jwt.MapClaims and the cast below fails.
		NewClaimsFunc: func(c *echo.Context) jwt.Claims {
			return &model.JwtPlayerClaims{}
		},
	}))

	api.GET("/test", func(ctx *echo.Context) error {
		return ctx.String(http.StatusOK, "Hello")
	})
	api.GET("/games", h.handleListGames)
	api.GET("/shared", h.handleGetShared)
	api.GET("/interactions", h.handleListInteractions)
	api.POST("/save", h.handleSave)
	api.POST("/play", h.handleResume)
	api.POST("/pause", h.handlePause)
}

func (h *HTTPAdapter) handleLogin(ctx *echo.Context) error {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := ctx.Bind(&body); err != nil {
		return badRequest(ctx, "Missing username or password")
	}

	res, err := h.d.Auth.LoginPlayer(ctx.Request().Context(), body.Username, body.Password)
	switch {
	case errors.Is(err, usecase.ErrBadRequest):
		return badRequest(ctx, "Missing username or password")
	case errors.Is(err, usecase.ErrUnauthorized):
		return unauthorised(ctx, "Incorrect username or password")
	case err != nil:
		log.Printf("login error: %v", err)
		return internal(ctx)
	}

	if h.tp == nil {
		log.Printf("time provider is not initialized")
		return internal(ctx)
	}
	now, err := h.tp.Time()
	if err != nil {
		log.Printf("failed to get current time: %v", err)
		return internal(ctx)
	}

	ctx.SetCookie(&http.Cookie{
		Name:     SessionCookieName,
		Value:    res.Token,
		Expires:  now.Add(constant.SessionLifetime),
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return ctx.NoContent(http.StatusNoContent)
}

// gameIDFromQuery parses and validates the required `gameId` query parameter.
func gameIDFromQuery(c *echo.Context) (int64, error) {
	raw := c.QueryParam("gameId")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid gameId %q: %w", raw, err)
	}
	return id, nil
}

// playerID returns the authenticated player ID from the JWT claims.
func (h *HTTPAdapter) playerID(c *echo.Context) (int64, bool) {
	token, ok := c.Get("user").(*jwt.Token)
	if !ok {
		return 0, false
	}
	claims, ok := token.Claims.(*model.JwtPlayerClaims)
	if !ok || claims.UserID < 1 {
		return 0, false
	}
	return claims.UserID, true
}

// accessChecked parses gameId, enforces the player's access, and returns the
// actor's player ID. Access failures write the response and return errStop so
// handlers stop; any response-write error is returned as-is.
func (h *HTTPAdapter) accessChecked(c *echo.Context) (gameID, playerID int64, err error) {
	gameID, err = gameIDFromQuery(c)
	if err != nil {
		log.Printf("bad request: %v", err)
		return 0, 0, badRequest(c, "Invalid gameId")
	}

	pid, ok := h.playerID(c)
	if !ok {
		return 0, 0, unauthorised(c, "Authentication required")
	}

	granted, err := h.d.Access.CheckPlayerAccess(c.Request().Context(), gameID, pid)
	if err != nil {
		log.Printf("access check error for game %d: %v", gameID, err)
		return 0, 0, internal(c)
	}
	if !granted {
		return 0, 0, notFound(c, "Game not found or inaccessible")
	}
	return gameID, pid, nil
}

func (h *HTTPAdapter) handleListGames(ctx *echo.Context) error {
	playerID, ok := h.playerID(ctx)
	if !ok {
		return unauthorised(ctx, "Authentication required")
	}

	games, err := h.d.Fetch.ListPlayerGames(ctx.Request().Context(), playerID)
	if err != nil {
		log.Printf("list games error: %v", err)
		return internal(ctx)
	}

	dtos := make([]gameDTO, 0, len(games))
	for _, g := range games {
		dtos = append(dtos, gameDTO{ID: g.ID, Name: g.Name})
	}
	return ctx.JSON(http.StatusOK, dtos)
}

func (h *HTTPAdapter) handleGetShared(ctx *echo.Context) error {
	gameID, _, err := h.accessChecked(ctx)
	if err != nil {
		return err
	}

	shared, err := h.d.Fetch.GetSharedData(ctx.Request().Context(), gameID)
	if err != nil {
		log.Printf("get shared data error: %v", err)
		return internal(ctx)
	}

	return ctx.JSON(http.StatusOK, sharedDataDTO{
		GameID:       shared.GameID,
		Status:       string(shared.Status),
		Valid:        shared.Valid,
		DeadlineAt:   shared.DeadlineAt,
		LastSavedAt:  shared.LastSavedAt,
		LastPausedAt: shared.LastPausedAt,
	})
}

func (h *HTTPAdapter) handleSave(ctx *echo.Context) error {
	gameID, playerID, err := h.accessChecked(ctx)
	if err != nil {
		return err
	}

	var body saveRequest
	if err := ctx.Bind(&body); err != nil {
		return badRequest(ctx, "Invalid request body")
	}
	if body.Duration < 1 {
		return badRequest(ctx, "duration must be >= 1")
	}

	if err := h.d.Commands.SaveGame(ctx.Request().Context(), gameID, playerID, body.Duration); err != nil {
		return commandError(ctx, err, "save game error", "Game is not currently playing")
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (h *HTTPAdapter) handleResume(ctx *echo.Context) error {
	gameID, playerID, err := h.accessChecked(ctx)
	if err != nil {
		return err
	}

	if err := h.d.Commands.ResumeGame(ctx.Request().Context(), gameID, playerID); err != nil {
		return commandError(ctx, err, "resume game error", "Game is not currently paused")
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (h *HTTPAdapter) handlePause(ctx *echo.Context) error {
	gameID, playerID, err := h.accessChecked(ctx)
	if err != nil {
		return err
	}

	if err := h.d.Commands.PauseGame(ctx.Request().Context(), gameID, playerID); err != nil {
		return commandError(ctx, err, "pause game error", "Game is not currently playing")
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (h *HTTPAdapter) handleListInteractions(ctx *echo.Context) error {
	gameID, _, err := h.accessChecked(ctx)
	if err != nil {
		return err
	}

	limit, err := interactionsLimit(ctx)
	if err != nil {
		log.Printf("bad request: %v", err)
		return badRequest(ctx, "Invalid limit")
	}

	interactions, err := h.d.Fetch.ListInteractions(ctx.Request().Context(), gameID, limit)
	if err != nil {
		log.Printf("list interactions error: %v", err)
		return internal(ctx)
	}

	dtos := make([]interactionDTO, 0, len(interactions))
	for _, i := range interactions {
		dtos = append(dtos, interactionDTO{
			ID:         i.ID,
			GameID:     i.GameID,
			PlayerID:   nullableInt64(i.PlayerID),
			Action:     i.Action,
			OccurredAt: i.OccurredAt,
			SavedBy:    nullableInt64(i.SavedBy),
		})
	}
	return ctx.JSON(http.StatusOK, dtos)
}

// interactionsLimit returns the `limit` query param, defaulting to
// defaultInteractionsLimit. Values below 1 are rejected.
func interactionsLimit(c *echo.Context) (int64, error) {
	raw := c.QueryParam("limit")
	if raw == "" {
		return defaultInteractionsLimit, nil
	}
	limit, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid limit %q: %w", raw, err)
	}
	if limit < 1 {
		return 0, errors.New("limit must be >= 1")
	}
	return limit, nil
}

func (h *HTTPAdapter) Run(ctx context.Context) error {
	e := echo.New()
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())
	h.routes(e)

	srv := &http.Server{Addr: h.addr, Handler: e}
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		log.Printf("Server stopping at %v...", h.addr)
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return <-errCh
	case err := <-errCh:
		return err
	}
}

// --- Response helpers ---

func errorJSON(c *echo.Context, status int, message string) error {
	return c.JSON(status, map[string]string{"message": message})
}

func badRequest(c *echo.Context, message string) error {
	return errorJSON(c, http.StatusBadRequest, message)
}

func unauthorised(c *echo.Context, message string) error {
	return errorJSON(c, http.StatusUnauthorized, message)
}

func notFound(c *echo.Context, message string) error {
	return errorJSON(c, http.StatusNotFound, message)
}

func internal(c *echo.Context) error {
	return errorJSON(c, http.StatusInternalServerError, "Something went wrong!")
}

// commandError maps a GameCommands error to a response: domain state-machine
// violations become 409, everything else becomes a logged 500.
func commandError(c *echo.Context, err error, logMsg, conflictMsg string) error {
	if conflictStatusFromErr(err) {
		return errorJSON(c, http.StatusConflict, conflictMsg)
	}
	log.Printf("%s: %v", logMsg, err)
	return internal(c)
}

// conflictStatusFromErr detects SQLite state-machine trigger violations.
func conflictStatusFromErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "cannot save while paused") ||
		strings.Contains(msg, "cannot pause") ||
		strings.Contains(msg, "cannot resume")
}

// nullableInt64 converts a database.NullInt64 into a *int64 (nil when invalid)
// so nullable IDs serialize as JSON null rather than an object.
func nullableInt64(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	x := v.Int64
	return &x
}