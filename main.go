package main

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"markovchain-chatbot/chatbot"
	"markovchain-chatbot/database"
	"markovchain-chatbot/discord"
	"markovchain-chatbot/filter"
	"markovchain-chatbot/markov"
	"markovchain-chatbot/settings"

	"github.com/lmittmann/tint"
	"github.com/mattn/go-isatty"
)

func main() {
	slog.SetDefault(slog.New(tint.NewHandler(os.Stderr, &tint.Options{
		Level:      slog.LevelDebug,
		TimeFormat: time.Kitchen,
		NoColor:    !isatty.IsTerminal(os.Stderr.Fd()),
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

	dbPath := fmt.Sprintf("%s_%s_markovchain.db", cfg.BotUsername, cfg.ChannelName)
	db, err := database.New(dbPath)
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	markovChain := markov.NewGenerator(db, cfg.BlacklistedWords, cfg.MaxSentenceWords)

	bot := chatbot.New(cfg, markovChain)

	if !cfg.TrainingMode && cfg.AutoGenerateMessages {
		for {
			time.Sleep(time.Duration(cfg.AutoGenerateInterval) * time.Second)
			message := markovChain.GenerateMessage()
			if filter.IsCleanMessage(message, cfg.AllowNonAsciiMessages) {
				bot.SendMessage(message)
			}
		}
	}

	fmt.Println("Press Enter to exit...")
	fmt.Scanln()
}
