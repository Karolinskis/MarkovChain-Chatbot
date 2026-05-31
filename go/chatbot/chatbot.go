package chatbot

import (
	"fmt"
	"log/slog"
	"strings"

	"markovchain-chatbot/discord"
	"markovchain-chatbot/markov"
	"markovchain-chatbot/settings"
	"markovchain-chatbot/tokenizer"

	twitch "github.com/gempir/go-twitch-irc/v4"
)

type Chatbot struct {
	client      *twitch.Client
	markovChain *markov.Generator
	cfg         *settings.Settings
	channelName string
	botUsername string
}

func New(cfg *settings.Settings, markovChain *markov.Generator) *Chatbot {
	client := twitch.NewClient(cfg.BotUsername, "oauth:"+strings.TrimPrefix(cfg.AccessToken, "oauth:"))

	bot := &Chatbot{
		client:      client,
		markovChain: markovChain,
		cfg:         cfg,
		channelName: cfg.ChannelName,
		botUsername: strings.ToLower(cfg.BotUsername),
	}

	client.OnConnect(func() {
		slog.Info("connected to Twitch")
	})

	client.OnSelfJoinMessage(func(message twitch.UserJoinMessage) {
		slog.Info("joined channel", "channel", message.Channel)
	})

	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		bot.onMessageReceived(message)
	})

	client.Join(cfg.ChannelName)

	go func() {
		if err := client.Connect(); err != nil {
			slog.Error("failed to connect", "error", err)
		}
	}()

	return bot
}

func (b *Chatbot) SendMessage(message string) {
	if strings.TrimSpace(message) == "" {
		return
	}

	slog.Info("sending message", "message", message)
	discord.Notify(message)
	b.client.Say(b.channelName, message)
}

func (b *Chatbot) onMessageReceived(message twitch.PrivateMessage) {
	if strings.EqualFold(message.User.Name, b.botUsername) {
		return
	}

	if b.cfg.IsUserBlocked(message.User.Name) {
		return
	}

	trimmedMsg := strings.TrimSpace(message.Message)

	if strings.EqualFold(trimmedMsg, "!stats") {
		stats := b.markovChain.GetStatistics()
		statsMessage := fmt.Sprintf("Dataset Statistics: Start Pairs: %d, Grammar Entries: %d",
			stats["TotalStartPairs"], stats["TotalGrammarEntries"])
		b.client.Reply(b.channelName, message.ID, statsMessage)
		return
	}

	if b.cfg.AllowGenerateCommand {
		for _, cmd := range b.cfg.GenerateCommands {
			if strings.HasPrefix(strings.ToLower(trimmedMsg), strings.ToLower(cmd)) {
				if b.cfg.IsUserAllowed(message.User.Name) {
					generatedMessage := b.markovChain.GenerateMessage()
					if generatedMessage != "" {
						b.client.Reply(b.channelName, message.ID, generatedMessage)
					}
				} else {
					slog.Info("generate command denied", "user", message.User.Name)
				}
				return
			}
		}
	}

	slog.Debug("message received", "user", message.User.Name, "message", message.Message)

	tokens := tokenizer.Tokenize(message.Message)
	if err := b.markovChain.Train(tokens); err != nil {
		slog.Error("failed to train", "error", err)
	}
}
