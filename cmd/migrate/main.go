package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/AndreeJait/zora-core/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const (
	migrationsPath = "files/migrations"
	usage          = `Usage: go run ./cmd/migrate <command>

Commands:
  up     Run all pending migrations
  down   Roll back the last migration
  fresh  Force down all migrations then re-run up`
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	command := os.Args[1]

	cfg, err := config.Load("files/config/app.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	m, err := migrate.New(
		"file://"+migrationsPath,
		cfg.DB.DSN,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create migrator: %v\n", err)
		os.Exit(1)
	}
	defer m.Close()

	switch command {
	case "up":
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			fmt.Fprintf(os.Stderr, "migration up failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("migration up completed successfully")

	case "down":
		if err := m.Steps(-1); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			fmt.Fprintf(os.Stderr, "migration down failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("migration down completed successfully")

	case "fresh":
		// Force down all migrations by repeatedly rolling back
		for {
			if err := m.Steps(-1); err != nil {
				break
			}
		}
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			fmt.Fprintf(os.Stderr, "migration fresh (up) failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("migration fresh completed successfully")

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n%s", command, usage)
		os.Exit(1)
	}
}