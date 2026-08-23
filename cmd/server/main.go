package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"keep-it-up/internal/application/usecase"
	"keep-it-up/internal/infrastructure/config"
	"keep-it-up/internal/infrastructure/database"
	"keep-it-up/internal/infrastructure/driven"
	"keep-it-up/internal/infrastructure/driver/httpadapter"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"

	_ "modernc.org/sqlite"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Printf("failed to load configuration: %v", err)
		return 1
	}

	sqlDB, err := sql.Open("sqlite", fmt.Sprintf(
		"file:%s?mode=rw",
		cfg.DBString,
	))
	if err != nil {
		fmt.Fprintf(os.Stderr, "open database: %v\n", err)
		return 1
	}
	defer func() {
		if err := sqlDB.Close(); err != nil {
			log.Printf("failed to close database connection: %v", err)
		}
	}()
	if err := sqlDB.PingContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "connect database: %v\n", err)
		return 1
	}

	q := database.New(sqlDB)

	timeProvider := &driven.DefaultTimeProvider{}

	fetching := usecase.NewDataFetching(q, timeProvider)
	commands := usecase.NewGameCommands(q, timeProvider)
	access := usecase.NewAccessManagement(q)

	deps := httpadapter.Deps{
		Auth: usecase.NewAuthentication(
			q,
			&driven.JwtTokenGenerator{
				JwtSecret:    cfg.JWTSecret,
				TimeProvider: timeProvider,
			},
		),
		Fetch:    fetching,
		Commands: commands,
		Access:   access,
	}

	adapter := httpadapter.New(
		cfg.ServerAddress,
		cfg.JWTSecret,
		timeProvider,
		deps,
	)

	var wg sync.WaitGroup
	errCh := make(chan error, 1)

	wg.Go(func() {
		log.Printf("Starting server on %s...\n", cfg.ServerAddress)
		if err := adapter.Run(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			wrappedErr := fmt.Errorf(
				"server failed to start or stopped unexpectedly: %w",
				err,
			)
			errCh <- wrappedErr
			stop()
			return
		}

		log.Println("server stopped gracefully")
	})

	<-ctx.Done()
	log.Printf("Shutdown requested: %v", ctx.Err())

	wg.Wait()
	close(errCh)

	select {
	case err := <-errCh:
		if err != nil {
			log.Printf("shutdown after server failure: %v", err)
			return 1
		}
	default:
	}

	return 0
}