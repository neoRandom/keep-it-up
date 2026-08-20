package main

import (
	"github.com/labstack/echo"
	"net/http"
)

func main() {
	e := echo.New()

	e.GET("/", func(ctx echo.Context) error {
		return ctx.JSON(
			http.StatusNoContent,
			map[string]string{"message": "Not implemented yet!"},
		)
	})

	if err := e.Start(":8080"); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}
