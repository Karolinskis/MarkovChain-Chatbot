package chatbot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"markovchain-chatbot/database"
	"markovchain-chatbot/settings"

	twitch "github.com/gempir/go-twitch-irc/v4"
)

// LiveChecker returns a map of channel name to live status.
// A nil LiveChecker treats all channels as always live.
type LiveChecker func(channels []string) (map[string]bool, error)

type Bot struct {
	client      *twitch.Client
	channels    map[string]*channel
	botUsername string
}

func New(ctx context.Context, cfg settings.BotConfig, db *database.Database, live LiveChecker) (*Bot, error) {
	accessToken := cfg.AccessToken
	if !strings.HasPrefix(accessToken, "oauth:") {
		accessToken = "oauth:" + accessToken
	}
	client := twitch.NewClient(cfg.BotUsername, accessToken)

	bot := &Bot{
		client:      client,
		channels:    make(map[string]*channel, len(cfg.Channels)),
		botUsername: strings.ToLower(cfg.BotUsername),
	}

	for _, chCfg := range cfg.Channels {
		channelID, err := db.EnsureChannel(ctx, cfg.BotUsername, chCfg.ChannelName)
		if err != nil {
			return nil, fmt.Errorf("ensure channel %s: %w", chCfg.ChannelName, err)
		}
		ch := newChannel(chCfg, channelID, client, db)
		if live == nil {
			ch.isLive.Store(true)
		}
		bot.channels[chCfg.ChannelName] = ch
	}

	client.OnConnect(func() {
		slog.Info("connected to Twitch IRC", "bot", cfg.BotUsername)
	})

	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		if ch, ok := bot.channels[message.Channel]; ok {
			ch.onMessage(bot.botUsername, message)
		}
	})

	client.OnClearMessage(func(message twitch.ClearMessage) {
		if ch, ok := bot.channels[message.Channel]; ok {
			ch.onDelete(message)
		}
	})

	for name, ch := range bot.channels {
		client.Join(name)
		if !ch.cfg.TrainingMode && ch.cfg.AutoGenerateMessages {
			go ch.startAutoGenerate(ctx)
		}
	}

	go func() {
		if err := client.Connect(); err != nil {
			slog.Error("IRC connection error", "bot", cfg.BotUsername, "error", err)
		}
	}()

	return bot, nil
}
