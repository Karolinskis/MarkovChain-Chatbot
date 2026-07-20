package chatbot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"markovchain-chatbot/internal/database"
	"markovchain-chatbot/internal/metrics"
	"markovchain-chatbot/internal/settings"

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

// New builds a bot and registers its message handlers. Call Run to connect.
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
		metrics.IRCUp.WithLabelValues(bot.botUsername).Set(1)
	})

	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		if ch, ok := bot.channels[message.Channel]; ok {
			ch.onMessage(ctx, bot.botUsername, message)
		}
	})

	client.OnClearMessage(func(message twitch.ClearMessage) {
		if ch, ok := bot.channels[message.Channel]; ok {
			ch.onDelete(ctx, message)
		}
	})

	return bot, nil
}

// Run joins the configured channels, connects to Twitch IRC, and starts the
// per-channel auto-generate loops. It blocks until ctx is cancelled or the
// connection fails, and waits for all goroutines it started to finish.
func (b *Bot) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	defer wg.Wait()
	defer cancel()
	defer metrics.IRCUp.WithLabelValues(b.botUsername).Set(0)

	for name, ch := range b.channels {
		b.client.Join(name)
		if !ch.cfg.TrainingMode && ch.cfg.AutoGenerateMessages {
			wg.Go(func() {
				ch.autoGenerate(ctx)
			})
		}
	}

	connectErr := make(chan error, 1)
	go func() {
		connectErr <- b.client.Connect()
	}()

	select {
	case <-ctx.Done():
		if err := b.client.Disconnect(); err != nil && !errors.Is(err, twitch.ErrConnectionIsNotOpen) {
			return fmt.Errorf("disconnect %s: %w", b.botUsername, err)
		}
		return nil
	case err := <-connectErr:
		return fmt.Errorf("connect as %s: %w", b.botUsername, err)
	}
}
