package httpadapter

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"keep-it-up/internal/infrastructure/constant"

	"github.com/labstack/echo/v5"
)

// IdempotencyStore is the Driven Adapter port required by the HTTP adapter
// (Driver layer) to persist idempotency state for mutation requests.
//
// The contract is expressed in primitive types so the concrete adapter can
// satisfy it without importing this package. `status` is one of
// constant.IdempotencyStatusInProgress / constant.IdempotencyStatusCompleted,
// or "" when the key is absent.
type IdempotencyStore interface {
	// Acquire atomically records key as IN_PROGRESS with the given TTL only if
	// the key does not already exist. It returns true when the caller won the
	// slot (first request for this key) and false when the key already exists.
	Acquire(ctx context.Context, key string, ttl time.Duration) (bool, error)

	// State returns the current status of key and, when COMPLETED, the cached
	// HTTP response (status code and body) to replay.
	State(ctx context.Context, key string) (status string, statusCode int, body []byte, err error)

	// Complete atomically transitions key to COMPLETED, storing the response
	// status code and body, and (re)sets the TTL.
	Complete(ctx context.Context, key string, statusCode int, body []byte, ttl time.Duration) error
}

// idempotency is the middleware that enforces at-most-once execution for
// mutation requests carrying an Idempotency-Key header. Requests without the
// header (or without a configured store) pass through untouched.
func (h *HTTPAdapter) idempotency(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		if h.idem == nil {
			return next(c)
		}

		rawKey := c.Request().Header.Get(h.idemHeader)
		if rawKey == "" {
			return next(c)
		}
		if !validIdempotencyKey(rawKey) {
			return badRequest(c, msgInvalidIdempotencyKey)
		}

		storeKey := h.idempotencyStoreKey(c, rawKey)
		ctx := c.Request().Context()

		acquired, err := h.idem.Acquire(ctx, storeKey, h.idemTTL)
		if err != nil {
			log.Printf("idempotency acquire error: %v", err)
			return internal(c)
		}

		if !acquired {
			status, code, body, err := h.idem.State(ctx, storeKey)
			if err != nil {
				log.Printf("idempotency state error: %v", err)
				return internal(c)
			}
			switch status {
			case constant.IdempotencyStatusInProgress:
				return errorJSON(c, http.StatusConflict, msgIdempotencyInProgress)
			case constant.IdempotencyStatusCompleted:
				return replayIdempotentResponse(c, code, body)
			default:
				log.Printf("idempotency: unexpected state %q for key %s", status, storeKey)
				return internal(c)
			}
		}

		rec := &recordingWriter{ResponseWriter: c.Response()}
		c.SetResponse(rec)

		if err := next(c); err != nil {
			return err
		}

		if err := h.idem.Complete(ctx, storeKey, rec.statusCode(), rec.body(), h.idemTTL); err != nil {
			// The response has already been written; a failed Complete leaves the
			// key IN_PROGRESS so a retry gets 409 rather than re-executing.
			log.Printf("idempotency complete error: %v", err)
		}
		return nil
	}
}

// idempotencyStoreKey scopes a key to the authenticated actor so two players
// cannot collide on the same Idempotency-Key value. Unauthenticated requests
// (e.g. /login) fall back to an anonymous scope.
func (h *HTTPAdapter) idempotencyStoreKey(c *echo.Context, rawKey string) string {
	if playerID, ok := h.playerID(c); ok {
		return fmt.Sprintf("idempotency:user:%d:%s", playerID, rawKey)
	}
	return fmt.Sprintf("idempotency:anon:%s", rawKey)
}

// replayIdempotentResponse reproduces the originally cached response. Bodyless
// responses (e.g. 204) are replayed as NoContent; otherwise the cached body is
// served as JSON, matching the mutation endpoints this middleware protects.
func replayIdempotentResponse(c *echo.Context, code int, body []byte) error {
	if len(body) == 0 {
		return c.NoContent(code)
	}
	return c.Blob(code, echo.MIMEApplicationJSON, body)
}

// validIdempotencyKey checks the header value is non-empty, within a sane
// length, and restricted to an unambiguous character set.
func validIdempotencyKey(key string) bool {
	if key == "" || len(key) > 128 {
		return false
	}
	for _, r := range key {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_' || r == '.' || r == ':':
		default:
			return false
		}
	}
	return true
}

// recordingWriter wraps the underlying ResponseWriter to capture the status
// code and body written by a handler so they can be persisted on completion.
type recordingWriter struct {
	http.ResponseWriter
	status int
	buf    bytes.Buffer
}

func (w *recordingWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

// Unwrap exposes the underlying writer so http.ResponseController and Echo
// internals can reach the original *echo.Response (see Context.SetResponse).
func (w *recordingWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *recordingWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	_, _ = w.buf.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *recordingWriter) statusCode() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func (w *recordingWriter) body() []byte {
	return w.buf.Bytes()
}
