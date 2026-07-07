package main

import (
	"context"
	"database/sql"
	"flag"
	"log/slog"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"markovchain-chatbot/internal/settings"
)

const migrationsDir = "internal/database/migrations"

func main() {
	settingsPath := flag.String("settings", "settings.json", "path to settings file")
	flag.Parse()

	command := "up"
	if args := flag.Args(); len(args) > 0 {
		command = args[0]
	}

	cfg, err := settings.Load(*settingsPath)
	if err != nil {
		slog.Error("load settings", "error", err)
		os.Exit(1)
	}

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		slog.Error("open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		slog.Error("set dialect", "error", err)
		os.Exit(1)
	}

	if err := goose.RunContext(context.Background(), command, db, migrationsDir); err != nil {
		slog.Error("migration", "command", command, "error", err)
		os.Exit(1)
	}
}
