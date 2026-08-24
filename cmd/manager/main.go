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
	auth, err := usecase.NewAuthentication(
		q,
		&driven.JwtTokenGenerator{
			JwtSecret:       cfg.JWTSecret,
			TimeProvider:    timeProvider,
			SessionLifetime: cfg.SessionLifetime,
		},
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create authentication:", err)
		return 1
	}
	players, err := usecase.NewPlayerManagement(q, auth)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create player management:", err)
		return 1
	}
	games, err := usecase.NewGameManagement(q)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create game management:", err)
		return 1
	}
	access, err := usecase.NewAccessManagement(q)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create access management:", err)
		return 1
	}
	fetch, err := usecase.NewDataFetching(q, timeProvider)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create data fetching:", err)
		return 1
	}
	commands, err := usecase.NewGameCommands(q, timeProvider)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create game commands:", err)
		return 1
	}

	deps := cliadapter.Deps{
		Games:    games,
		Access:   access,
		Players:  players,
		Auth:     auth,
		Fetch:    fetch,
		Commands: commands,
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
