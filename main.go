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

	if !cfg.TrainingMode && cfg.AutoGenerateMessages {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Duration(cfg.AutoGenerateInterval) * time.Second):
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
