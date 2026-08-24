package httpadapter

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"keep-it-up/internal/infrastructure/driven"
	"keep-it-up/internal/application/usecase"
	"keep-it-up/internal/infrastructure/constant"
	coremodel "keep-it-up/internal/core/model"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"
)

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

func (h *HTTPAdapter) handleListGames(ctx *echo.Context) error {
	playerID, ok := h.playerID(ctx)
	if !ok {
		return unauthorised(ctx, msgAuthenticationRequired)
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
	gameID, _, denied, err := h.accessChecked(ctx)
	if err != nil {
		return err
	}
	if denied {
		return nil
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
	gameID, playerID, denied, err := h.accessChecked(ctx)
	if err != nil {
		return err
	}
	if denied {
		return nil
	}

	var body saveRequest
	if err := ctx.Bind(&body); err != nil {
		return badRequest(ctx, "Invalid request body")
	}
	if body.Duration < 1 {
		return badRequest(ctx, "duration must be >= 1")
	}

	if err := h.d.Commands.SaveGame(ctx.Request().Context(), gameID, playerID, body.Duration); err != nil {
		return commandError(ctx, err, "save game error", msgGameNotPlaying)
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (h *HTTPAdapter) handleResume(ctx *echo.Context) error {
	gameID, playerID, denied, err := h.accessChecked(ctx)
	if err != nil {
		return err
	}
	if denied {
		return nil
	}

	if err := h.d.Commands.ResumeGame(ctx.Request().Context(), gameID, playerID); err != nil {
		return commandError(ctx, err, "resume game error", msgGameNotPaused)
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (h *HTTPAdapter) handlePause(ctx *echo.Context) error {
	gameID, playerID, denied, err := h.accessChecked(ctx)
	if err != nil {
		return err
	}
	if denied {
		return nil
	}

	if err := h.d.Commands.PauseGame(ctx.Request().Context(), gameID, playerID); err != nil {
		return commandError(ctx, err, "pause game error", msgGameNotPlaying)
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (h *HTTPAdapter) handleListInteractions(ctx *echo.Context) error {
	selector, err := querySelector(ctx)
	if err != nil {
		log.Printf("bad request: %v", err)
		return badRequest(ctx, "Invalid query")
	}

	limit, err := interactionsLimit(ctx)
	if err != nil {
		log.Printf("bad request: %v", err)
		return badRequest(ctx, "Invalid limit")
	}
	offset, err := interactionsOffset(ctx)
	if err != nil {
		log.Printf("bad request: %v", err)
		return badRequest(ctx, "Invalid offset")
	}

	gameID, playerID, denied, writeErr := h.accessChecked(ctx)
	if writeErr != nil {
		return writeErr
	}
	if denied {
		return nil
	}

	cctx := ctx.Request().Context()

	switch selector {
	case queryAll:
		interactions, err := h.d.Fetch.ListInteractions(cctx, gameID, limit, offset)
		if err != nil {
			log.Printf("list interactions error: %v", err)
			return internal(ctx)
		}
		return ctx.JSON(http.StatusOK, toInteractionDTOs(interactions))

	case queryPlayer:
		interactions, err := h.d.Fetch.ListPlayerInteractions(cctx, gameID, playerID, limit, offset)
		if err != nil {
			log.Printf("list player interactions error: %v", err)
			return internal(ctx)
		}
		return ctx.JSON(http.StatusOK, toInteractionDTOs(interactions))

	case queryFirst:
		interaction, err := h.d.Fetch.FirstInteraction(cctx, gameID)
		if err != nil {
			log.Printf("get first interaction error: %v", err)
			return internal(ctx)
		}
		return ctx.JSON(http.StatusOK, interactionDTOFrom(interaction))

	default: // queryLast
		interaction, err := h.d.Fetch.LastInteraction(cctx, gameID)
		if err != nil {
			log.Printf("get last interaction error: %v", err)
			return internal(ctx)
		}
		return ctx.JSON(http.StatusOK, interactionDTOFrom(interaction))
	}
}

// toInteractionDTOs maps interaction rows to client-facing DTOs.
func toInteractionDTOs(interactions []coremodel.Interaction) []interactionDTO {
	dtos := make([]interactionDTO, 0, len(interactions))
	for _, i := range interactions {
		dtos = append(dtos, interactionDTO{
			ID:         i.ID,
			GameID:     i.GameID,
			PlayerID:   i.PlayerID,
			Action:     i.Action,
			OccurredAt: i.OccurredAt.Format(constant.DBDatetimeFormat),
			SavedBy:    i.SavedBy,
		})
	}
	return dtos
}

// interactionDTOFrom maps a single interaction (or nil) to a DTO, letting
// first/last return JSON null when a game has no interactions.
func interactionDTOFrom(i *coremodel.Interaction) *interactionDTO {
	if i == nil {
		return nil
	}
	return &interactionDTO{
		ID:         i.ID,
		GameID:     i.GameID,
		PlayerID:   i.PlayerID,
		Action:     i.Action,
		OccurredAt: i.OccurredAt.Format(constant.DBDatetimeFormat),
		SavedBy:    i.SavedBy,
	}
}

// playerID returns the authenticated player ID from the JWT claims.
func (h *HTTPAdapter) playerID(c *echo.Context) (int64, bool) {
	token, ok := c.Get("user").(*jwt.Token)
	if !ok {
		return 0, false
	}
	claims, ok := token.Claims.(*driven.JwtClaims)
	if !ok || claims.UserID < 1 {
		return 0, false
	}
	return claims.UserID, true
}

// accessChecked parses gameId, verifies the player's access, and returns the
// actor's player ID. When access is denied it writes the response and returns
// denied=true so the handler stops; writeErr carries any failure to write it.
func (h *HTTPAdapter) accessChecked(c *echo.Context) (gameID, playerID int64, denied bool, writeErr error) {
	gameID, err := gameIDFromQuery(c)
	if err != nil {
		log.Printf("bad request: %v", err)
		return 0, 0, true, badRequest(c, msgInvalidGameID)
	}

	playerID, ok := h.playerID(c)
	if !ok {
		return 0, 0, true, unauthorised(c, msgAuthenticationRequired)
	}

	granted, err := h.d.Access.CheckPlayerAccess(c.Request().Context(), gameID, playerID)
	if err != nil {
		log.Printf("access check error for game %d: %v", gameID, err)
		return 0, 0, true, internal(c)
	}
	if !granted {
		return 0, 0, true, notFound(c, msgGameNotFound)
	}
	return gameID, playerID, false, nil
}

// gameIDFromQuery parses the required `gameId` query parameter.
func gameIDFromQuery(c *echo.Context) (int64, error) {
	raw := c.QueryParam("gameId")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid gameId %q: %w", raw, err)
	}
	return id, nil
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

// interactionsOffset returns the `offset` query param, defaulting to
// defaultInteractionsOffset. Values below 0 are rejected.
func interactionsOffset(c *echo.Context) (int64, error) {
	raw := c.QueryParam("offset")
	if raw == "" {
		return defaultInteractionsOffset, nil
	}
	offset, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid offset %q: %w", raw, err)
	}
	if offset < 0 {
		return 0, errors.New("offset must be >= 0")
	}
	return offset, nil
}

// querySelector parses the `query` query param, defaulting to queryAll.
// It rejects any value outside the allowed enum.
func querySelector(c *echo.Context) (string, error) {
	raw := c.QueryParam("query")
	if raw == "" {
		return queryAll, nil
	}
	switch raw {
	case queryAll, queryPlayer, queryFirst, queryLast:
		return raw, nil
	default:
		return "", fmt.Errorf("invalid query %q", raw)
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

// commandError maps a GameCommands error to a response: state-machine
// violations become 409, everything else becomes a logged 500.
func commandError(c *echo.Context, err error, logMsg, conflictMsg string) error {
	if errors.Is(err, usecase.ErrCannotSaveWhilePaused) ||
		errors.Is(err, usecase.ErrCannotPause) ||
		errors.Is(err, usecase.ErrCannotResume) {
		return errorJSON(c, http.StatusConflict, conflictMsg)
	}
	log.Printf("%s: %v", logMsg, err)
	return internal(c)
}

