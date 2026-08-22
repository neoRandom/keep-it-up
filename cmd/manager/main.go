package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"keep-it-up/internal/application/usecase"
	"keep-it-up/internal/infrastructure/constant"
	"keep-it-up/internal/infrastructure/database"
	"keep-it-up/internal/infrastructure/driven"
	"keep-it-up/internal/infrastructure/driver/cliadapter"
	"keep-it-up/internal/infrastructure/util"

	_ "modernc.org/sqlite"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// --- Driven side: infrastructure dependencies ---------------------
	util.LoadEnv(constant.EnvFilename)

	dbString, err := filepath.Abs(os.Getenv("GOOSE_DBSTRING"))
	if err != nil {
		fmt.Printf("failed to get dbstring absolute path: %v\n", err)
		return 1
	}

	sqlDB, err := sql.Open("sqlite", fmt.Sprintf(
		"file:%s?mode=rw",
		dbString,
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
	auth := usecase.NewAuthentication(q, nil)
	deps := cliadapter.Deps{
		Games:    usecase.NewGameManagement(q),
		Access:   usecase.NewAccessManagement(q),
		Players:  usecase.NewPlayerManagement(q, auth),
		Auth:     auth,
		Fetch:    usecase.NewDataFetching(q),
		Commands: usecase.NewGameCommands(q, &driven.DefaultTimeProvider{}),
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
