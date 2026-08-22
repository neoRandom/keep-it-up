package httpadapter

import (
	"context"
	"database/sql"
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

// SessionCookieName is the cookie the API uses to carry the JWT, matching the
// `sessionCookie` security scheme in api/openapi.yaml.
const SessionCookieName string = "session"

type Deps struct {
	Auth     port.Authentication
	Fetch    port.DataFetching
	Commands port.GameCommands
	Access   port.AccessManagement
}

// gameDTO is the JSON representation of a game as documented in
// api/openapi.yaml (schema `Game`).
type gameDTO struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// sharedDataDTO is the JSON representation of a game's shared state as
// documented in api/openapi.yaml (schema `SharedData`).
type sharedDataDTO struct {
	GameID       int64      `json:"gameId"`
	Status       string     `json:"status"`
	Valid        *bool      `json:"valid"`
	DeadlineAt   *time.Time `json:"deadlineAt"`
	LastSavedAt  *time.Time `json:"lastSavedAt"`
	LastPausedAt *time.Time `json:"lastPausedAt"`
}

// interactionDTO is the JSON representation of an interaction as documented in
// api/openapi.yaml (schema `Interaction`).
type interactionDTO struct {
	ID         int64  `json:"id"`
	GameID     int64  `json:"gameId"`
	PlayerID   *int64 `json:"playerId"`
	Action     string `json:"action"`
	OccurredAt string `json:"occurredAt"`
	SavedBy    *int64 `json:"savedBy"`
}

// saveRequest is the JSON request body for POST /api/save, documented in
// api/openapi.yaml (schema `SaveRequest`).
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

// routes registers every HTTP route and its middleware on the provided Echo
// instance. It is separated from Run so handlers can be unit-tested with Echo's
// test helpers without opening a real listener.
func (h *HTTPAdapter) routes(e *echo.Echo) {
	unprotectedApi := e.Group("/api")

	unprotectedApi.POST("/login", h.handleLogin)

	api := unprotectedApi.Group("")
	api.Use(echojwt.WithConfig(echojwt.Config{
		SigningKey:  []byte(h.jwtSecret),
		TokenLookup: fmt.Sprintf("cookie:%s", SessionCookieName),
		// Parse the JWT into our typed claims so handlers can read the actor's
		// UserID directly. Without this the middleware defaults to jwt.MapClaims
		// and the *model.JwtPlayerClaims cast in playerIDFromContext fails.
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

// handleLogin parses the JSON body per api/openapi.yaml, authenticates the
// player, and sets the session cookie on success. It returns an empty 204.
func (h *HTTPAdapter) handleLogin(ctx *echo.Context) error {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := ctx.Bind(&body); err != nil {
		return ctx.JSON(
			http.StatusBadRequest,
			map[string]string{
				"message": "Missing username or password",
			},
		)
	}

	res, err := h.d.Auth.LoginPlayer(
		ctx.Request().Context(),
		body.Username, body.Password,
	)
	if err != nil {
		if err == usecase.ErrBadRequest {
			return ctx.JSON(
				http.StatusBadRequest,
				map[string]string{
					"message": "Missing username or password",
				},
			)
		}
		if err == usecase.ErrUnauthorized {
			return ctx.JSON(
				http.StatusUnauthorized,
				map[string]string{
					"message": "Incorrect username or password",
				},
			)
		}

		log.Printf("login error: %v", err)
		return ctx.JSON(
			http.StatusInternalServerError,
			map[string]string{
				"message": "Something went wrong!",
			},
		)
	}

	if h.tp == nil {
		log.Printf("time provider is not initialized")
		return ctx.JSON(
			http.StatusInternalServerError,
			map[string]string{
				"message": "Something went wrong!",
			},
		)
	}

	t, err := h.tp.Time()
	if err != nil {
		log.Printf("failed to get current time: %v", err)
		return ctx.JSON(
			http.StatusInternalServerError,
			map[string]string{
				"message": "Something went wrong!",
			},
		)
	}

	ctx.SetCookie(&http.Cookie{
		Name:     SessionCookieName,
		Value:    res.Token,
		Expires:  t.Add(constant.SessionLifetime),
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return ctx.NoContent(http.StatusNoContent)
}

// parseGameID extracts and parses the required `gameId` query parameter. The
// underlying ParseInt error is wrapped (not discarded) so callers can log the
// actual cause, matching the CLI adapter's parseID convention.
func parseGameID(c *echo.Context) (int64, error) {
	raw := c.QueryParam("gameId")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid gameId %q: %w", raw, err)
	}
	return id, nil
}

// playerIDFromContext returns the authenticated player ID from the JWT claims
// stored in the context by the echo-jwt middleware.
func (h *HTTPAdapter) playerIDFromContext(c *echo.Context) (int64, bool) {
	user := c.Get("user")
	token, ok := user.(*jwt.Token)
	if !ok {
		return 0, false
	}
	claims, ok := token.Claims.(*model.JwtPlayerClaims)
	if !ok {
		return 0, false
	}
	if claims.UserID < 1 {
		return 0, false
	}
	return claims.UserID, true
}

// requireAccess enforces that the authenticated player has access to the game.
// A player without access receives 404 (indistinguishable from "not found", per
// the spec's `GameNotFound` "or inaccessible" semantics). Any internal error
// becomes a 500; an unauthenticated request becomes a 401.
//
// It returns (denied, writeErr): denied is true when a 4xx/5xx response has been
// written and the handler must stop; writeErr is non-nil only if writing that
// response itself failed, which handlers propagate upward.
func (h *HTTPAdapter) requireAccess(c *echo.Context, gameID int64) (denied bool, writeErr error) {
	playerID, ok := h.playerIDFromContext(c)
	if !ok {
		return true, c.JSON(http.StatusUnauthorized, map[string]string{"message": "Authentication required"})
	}

	granted, err := h.d.Access.CheckPlayerAccess(c.Request().Context(), gameID, playerID)
	if err != nil {
		log.Printf("access check error for game %d: %v", gameID, err)
		return true, c.JSON(http.StatusInternalServerError, map[string]string{"message": "Something went wrong!"})
	}
	if !granted {
		return true, c.JSON(http.StatusNotFound, map[string]string{"message": "Game not found or inaccessible"})
	}
	return false, nil
}

// nullableInt64 converts a database.NullInt64 into a *int64 (nil when invalid)
// so nullable IDs are serialized as JSON null rather than an object.
func nullableInt64(v sql.NullInt64) *int64 {
	if !v.Valid {
		return nil
	}
	x := v.Int64
	return &x
}

func (h *HTTPAdapter) handleListGames(ctx *echo.Context) error {
	playerID, ok := h.playerIDFromContext(ctx)
	if !ok {
		return ctx.JSON(http.StatusUnauthorized, map[string]string{"message": "Authentication required"})
	}

	games, err := h.d.Fetch.ListPlayerGames(ctx.Request().Context(), playerID)
	if err != nil {
		log.Printf("list games error: %v", err)
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"message": "Something went wrong!"})
	}

	dtos := make([]gameDTO, 0, len(games))
	for _, g := range games {
		dtos = append(dtos, gameDTO{ID: g.ID, Name: g.Name})
	}
	return ctx.JSON(http.StatusOK, dtos)
}

func (h *HTTPAdapter) handleGetShared(ctx *echo.Context) error {
	gameID, err := parseGameID(ctx)
	if err != nil {
		log.Printf("bad request: %v", err)
		return ctx.JSON(http.StatusBadRequest, map[string]string{"message": "Invalid gameId"})
	}
	denied, err := h.requireAccess(ctx, gameID)
	if err != nil {
		return err
	}
	if denied {
		return nil
	}

	shared, err := h.d.Fetch.GetSharedData(ctx.Request().Context(), gameID)
	if err != nil {
		log.Printf("get shared data error: %v", err)
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"message": "Something went wrong!"})
	}

	sharedDTO := sharedDataDTO{
		GameID:       shared.GameID,
		Status:       string(shared.Status),
		Valid:        shared.Valid,
		DeadlineAt:   shared.DeadlineAt,
		LastSavedAt:  shared.LastSavedAt,
		LastPausedAt: shared.LastPausedAt,
	}
	return ctx.JSON(http.StatusOK, sharedDTO)
}

// conflictStatusFromErr reports whether an error from the game commands layer is
// a domain state-machine violation that should surface as HTTP 409 Conflict
// (per the spec's save/play/pause 409 responses). The violations originate from
// the SQLite trigger `trg_interactions_state_machine`.
func conflictStatusFromErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "cannot save while paused") ||
		strings.Contains(msg, "cannot pause") ||
		strings.Contains(msg, "cannot resume")
}

func (h *HTTPAdapter) handleSave(ctx *echo.Context) error {
	gameID, err := parseGameID(ctx)
	if err != nil {
		log.Printf("bad request: %v", err)
		return ctx.JSON(http.StatusBadRequest, map[string]string{"message": "Invalid gameId"})
	}
	denied, err := h.requireAccess(ctx, gameID)
	if err != nil {
		return err
	}
	if denied {
		return nil
	}

	var body saveRequest
	if err := ctx.Bind(&body); err != nil {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"message": "Invalid request body"})
	}
	if body.Duration < 1 {
		return ctx.JSON(http.StatusBadRequest, map[string]string{"message": "duration must be >= 1"})
	}

	playerID, ok := h.playerIDFromContext(ctx)
	if !ok {
		return ctx.JSON(http.StatusUnauthorized, map[string]string{"message": "Authentication required"})
	}

	if err := h.d.Commands.SaveGame(ctx.Request().Context(), gameID, playerID, body.Duration); err != nil {
		if conflictStatusFromErr(err) {
			return ctx.JSON(http.StatusConflict, map[string]string{"message": "Game is not currently playing"})
		}
		log.Printf("save game error: %v", err)
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"message": "Something went wrong!"})
	}

	return ctx.NoContent(http.StatusNoContent)
}

func (h *HTTPAdapter) handleResume(ctx *echo.Context) error {
	gameID, err := parseGameID(ctx)
	if err != nil {
		log.Printf("bad request: %v", err)
		return ctx.JSON(http.StatusBadRequest, map[string]string{"message": "Invalid gameId"})
	}
	denied, err := h.requireAccess(ctx, gameID)
	if err != nil {
		return err
	}
	if denied {
		return nil
	}

	playerID, ok := h.playerIDFromContext(ctx)
	if !ok {
		return ctx.JSON(http.StatusUnauthorized, map[string]string{"message": "Authentication required"})
	}

	if err := h.d.Commands.ResumeGame(ctx.Request().Context(), gameID, playerID); err != nil {
		if conflictStatusFromErr(err) {
			return ctx.JSON(http.StatusConflict, map[string]string{"message": "Game is not currently paused"})
		}
		log.Printf("resume game error: %v", err)
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"message": "Something went wrong!"})
	}

	return ctx.NoContent(http.StatusNoContent)
}

func (h *HTTPAdapter) handlePause(ctx *echo.Context) error {
	gameID, err := parseGameID(ctx)
	if err != nil {
		log.Printf("bad request: %v", err)
		return ctx.JSON(http.StatusBadRequest, map[string]string{"message": "Invalid gameId"})
	}
	denied, err := h.requireAccess(ctx, gameID)
	if err != nil {
		return err
	}
	if denied {
		return nil
	}

	playerID, ok := h.playerIDFromContext(ctx)
	if !ok {
		return ctx.JSON(http.StatusUnauthorized, map[string]string{"message": "Authentication required"})
	}

	if err := h.d.Commands.PauseGame(ctx.Request().Context(), gameID, playerID); err != nil {
		if conflictStatusFromErr(err) {
			return ctx.JSON(http.StatusConflict, map[string]string{"message": "Game is not currently playing"})
		}
		log.Printf("pause game error: %v", err)
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"message": "Something went wrong!"})
	}

	return ctx.NoContent(http.StatusNoContent)
}

func (h *HTTPAdapter) handleListInteractions(ctx *echo.Context) error {
	gameID, err := parseGameID(ctx)
	if err != nil {
		log.Printf("bad request: %v", err)
		return ctx.JSON(http.StatusBadRequest, map[string]string{"message": "Invalid gameId"})
	}

	limit := int64(20) // spec default
	if raw := ctx.QueryParam("limit"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed < 1 {
			log.Printf("bad request: invalid limit %q: %v", raw, err)
			return ctx.JSON(http.StatusBadRequest, map[string]string{"message": "Invalid limit"})
		}
		limit = parsed
	}

	denied, err := h.requireAccess(ctx, gameID)
	if err != nil {
		return err
	}
	if denied {
		return nil
	}

	interactions, err := h.d.Fetch.ListInteractions(ctx.Request().Context(), gameID, limit)
	if err != nil {
		log.Printf("list interactions error: %v", err)
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"message": "Something went wrong!"})
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

func (h *HTTPAdapter) Run(ctx context.Context) error {
	e := echo.New()

	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	h.routes(e)

	srv := &http.Server{
		Addr:    h.addr,
		Handler: e,
	}

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