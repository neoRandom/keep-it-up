package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"keep-it-up/internal/application/usecase"
	"keep-it-up/internal/infrastructure/config"
	"keep-it-up/internal/infrastructure/database"
	"keep-it-up/internal/infrastructure/driven"
	"keep-it-up/internal/infrastructure/driver/cliadapter"

	_ "modernc.org/sqlite"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// --- Configuration -------------------------------------------------
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load configuration: %v\n", err)
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

	// --- Application side: use cases implementing the driver ports ----
	timeProvider := &driven.DefaultTimeProvider{}
	auth := usecase.NewAuthentication(q, nil)
	players, err := usecase.NewPlayerManagement(q, auth)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create player management: %v\n", err)
		return 1
	}

	deps := cliadapter.Deps{
		Games:    usecase.NewGameManagement(q),
		Access:   usecase.NewAccessManagement(q),
		Players:  players,
		Auth:     auth,
		Fetch:    usecase.NewDataFetching(q, timeProvider),
		Commands: usecase.NewGameCommands(q, timeProvider),
		Stdout:   os.Stdout,
		Stderr:   os.Stderr,
	}

	// --- Driver side: CLI adapter --------------------------------------
	cli := cliadapter.New(deps)

	if err := cli.Run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
