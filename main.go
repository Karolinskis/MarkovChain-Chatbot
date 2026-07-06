package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"markovchain-chatbot/chatbot"
	"markovchain-chatbot/database"
	"markovchain-chatbot/helix"
	"markovchain-chatbot/settings"

	"github.com/lmittmann/tint"
)

func main() {
	slog.SetDefault(slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: time.Kitchen,
	})))

	settingsPath := "settings.json"
	if len(os.Args) > 1 {
		settingsPath = os.Args[1]
	}

	cfg, err := settings.Load(settingsPath)
	if err != nil {
		slog.Error("failed to load settings", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	var live chatbot.LiveChecker
	if cfg.HelixClientID != "" && cfg.HelixClientSecret != "" {
		helixClient, err := helix.New(cfg.HelixClientID, cfg.HelixClientSecret)
		if err != nil {
			slog.Error("failed to create helix client", "error", err)
			os.Exit(1)
		}
		live = helixClient.LiveChannels
		slog.Info("live detection enabled")
	} else {
		slog.Warn("helix credentials not set, all channels treated as always live")
	}

	for _, botCfg := range cfg.Bots {
		if _, err := chatbot.New(ctx, botCfg, db, live); err != nil {
			slog.Error("failed to start bot", "bot", botCfg.BotUsername, "error", err)
			os.Exit(1)
		}
	}

	<-ctx.Done()
	slog.Info("shutting down")
}
