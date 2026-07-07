package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"markovchain-chatbot/internal/chatbot"
	"markovchain-chatbot/internal/database"
	"markovchain-chatbot/internal/helix"
	"markovchain-chatbot/internal/settings"

	"github.com/lmittmann/tint"
	"golang.org/x/sync/errgroup"
)

func main() {
	slog.SetDefault(slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: time.Kitchen,
	})))

	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	settingsPath := "settings.json"
	if len(os.Args) > 1 {
		settingsPath = os.Args[1]
	}

	cfg, err := settings.Load(settingsPath)
	if err != nil {
		return fmt.Errorf("load settings: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}
	defer db.Close()

	var live chatbot.LiveChecker
	if cfg.HelixClientID != "" && cfg.HelixClientSecret != "" {
		helixClient, err := helix.New(cfg.HelixClientID, cfg.HelixClientSecret)
		if err != nil {
			return fmt.Errorf("create helix client: %w", err)
		}
		live = helixClient.LiveChannels
		slog.Info("live detection enabled")
	} else {
		slog.Warn("helix credentials not set, all channels treated as always live")
	}

	bots := make([]*chatbot.Bot, 0, len(cfg.Bots))
	for _, botCfg := range cfg.Bots {
		bot, err := chatbot.New(ctx, botCfg, db, live)
		if err != nil {
			return fmt.Errorf("create bot %s: %w", botCfg.BotUsername, err)
		}
		bots = append(bots, bot)
	}

	g, ctx := errgroup.WithContext(ctx)
	for _, bot := range bots {
		g.Go(func() error {
			return bot.Run(ctx)
		})
	}
	if live != nil {
		g.Go(func() error {
			chatbot.RunLivePoller(ctx, live, bots)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}
	slog.Info("shut down")
	return nil
}
