package httpadapter

import (
	"context"
	"fmt"
	"keep-it-up/internal/application/usecase"
	"keep-it-up/internal/core/port"
	"log"
	"net/http"
	"time"

	echojwt "github.com/labstack/echo-jwt/v5"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

const JWTTokenCookieName string = "access_token"

type Deps struct {
	Auth port.Authentication
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
		tp: tp,
		d:         d,
	}
}

func (h *HTTPAdapter) Run(ctx context.Context) error {
	e := echo.New()

	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())

	unprotectedApi := e.Group("/api")

	unprotectedApi.POST("/login", func(ctx *echo.Context) error {
		username := ctx.FormValue("username")
		password := ctx.FormValue("password")

		res, err := h.d.Auth.LoginPlayer(
			ctx.Request().Context(),
			username, password,
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
			Name: JWTTokenCookieName,
			Value: res.Token,
			Expires: t.Add(24 * time.Hour),
			Path: "/",
			Secure: true,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})

		return ctx.JSON(
			http.StatusNoContent, 
			map[string]string{
				"message": "Login successful; authentication cookies are set.",
			},
		)
	})

	api := unprotectedApi.Group("")
	api.Use(echojwt.WithConfig(echojwt.Config{
		SigningKey: []byte(h.jwtSecret),
		TokenLookup: fmt.Sprintf("cookie:%s", JWTTokenCookieName),
	}))

	api.GET("/test", func(ctx *echo.Context) error {
		return ctx.String(http.StatusOK, "Hello")
	})

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
