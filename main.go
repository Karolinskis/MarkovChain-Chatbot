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
	"markovchain-chatbot/discord"
	"markovchain-chatbot/filter"
	"markovchain-chatbot/helix"
	"markovchain-chatbot/markov"
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

	if cfg.EnableDiscordLogging {
		discord.Init(cfg.DiscordWebhookURL)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	channelID, err := db.EnsureChannel(ctx, cfg.BotUsername, cfg.ChannelName)
	if err != nil {
		slog.Error("failed to ensure channel", "error", err)
		os.Exit(1)
	}

	markovChain := markov.NewGenerator(db, channelID, cfg.BlacklistedWords, cfg.MaxSentenceWords)

	bot := chatbot.New(cfg, db, markovChain, channelID)

	var helixClient *helix.Client
	if cfg.HelixClientID != "" && cfg.HelixClientSecret != "" {
		helixClient, err = helix.New(cfg.HelixClientID, cfg.HelixClientSecret)
		if err != nil {
			slog.Error("failed to create helix client", "error", err)
			os.Exit(1)
		}
		slog.Info("live detection enabled", "channel", cfg.ChannelName)
	} else {
		slog.Warn("helix credentials not set, auto-generate will run unconditionally")
	}

	if !cfg.TrainingMode && cfg.AutoGenerateMessages {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Duration(cfg.AutoGenerateInterval) * time.Second):
					if helixClient != nil {
						statuses, err := helixClient.LiveChannels([]string{cfg.ChannelName})
						if err != nil {
							slog.Warn("failed to check stream status", "error", err)
							continue
						}
						if !statuses[cfg.ChannelName] {
							slog.Debug("stream offline, skipping auto generate")
							continue
						}
					}
					message := markovChain.GenerateMessage(ctx)
					if filter.IsCleanMessage(message, cfg.AllowNonAsciiMessages) {
						bot.SendMessage(message)
					}
				}
			}
		}()
	}

	<-ctx.Done()
	slog.Info("shutting down")
}
