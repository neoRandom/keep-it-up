package httpadapter

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"keep-it-up/internal/application/model"
	"keep-it-up/internal/core/port"

	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v5"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

// Deps holds the ports the HTTP adapter drives.
type Deps struct {
	Auth     port.Authentication
	Fetch    port.DataFetching
	Commands port.GameCommands
	Access   port.AccessManagement
}

type HTTPAdapter struct {
	addr      string
	jwtSecret string
	tp        port.TimeProvider
	d         Deps

	idem       IdempotencyStore
	idemTTL    time.Duration
	idemHeader string
}

// Option configures an HTTPAdapter. Kept separate from the constructor so the
// idempotency wiring stays optional and existing call sites are unchanged.
type Option func(*HTTPAdapter)

// WithIdempotency enables idempotency enforcement using the given store, TTL,
// and header name. When omitted the adapter runs without idempotency handling.
func WithIdempotency(store IdempotencyStore, ttl time.Duration, header string) Option {
	return func(h *HTTPAdapter) {
		h.idem = store
		h.idemTTL = ttl
		h.idemHeader = header
	}
}

func New(addr string, jwtSecret string, tp port.TimeProvider, d Deps, opts ...Option) *HTTPAdapter {
	h := &HTTPAdapter{addr: addr, jwtSecret: jwtSecret, tp: tp, d: d}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// Run serves the HTTP API until ctx is cancelled, then shuts down gracefully.
func (h *HTTPAdapter) Run(ctx context.Context) error {
	e := echo.New()
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())
	h.routes(e)

	srv := &http.Server{Addr: h.addr, Handler: e}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

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

// routes wires middleware and handlers. Separate from Run so handlers can be
// unit-tested with Echo's test helpers.
func (h *HTTPAdapter) routes(e *echo.Echo) {
	idem := h.idempotency

	unprotected := e.Group("/api")
	unprotected.POST("/login", h.handleLogin, idem)

	api := unprotected.Group("")
	api.Use(echojwt.WithConfig(echojwt.Config{
		SigningKey:  []byte(h.jwtSecret),
		TokenLookup: fmt.Sprintf("cookie:%s", SessionCookieName),
		// Parse into typed claims so handlers can read the actor's UserID;
		// otherwise the middleware defaults to jwt.MapClaims and the cast fails.
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
	api.POST("/save", h.handleSave, idem)
	api.POST("/play", h.handleResume, idem)
	api.POST("/pause", h.handlePause, idem)
}
