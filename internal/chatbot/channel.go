package chatbot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	"markovchain-chatbot/internal/database"
	"markovchain-chatbot/internal/markov"
	"markovchain-chatbot/internal/settings"
	"markovchain-chatbot/internal/tokenizer"

	twitch "github.com/gempir/go-twitch-irc/v4"
)

type channel struct {
	id     int
	markov *markov.Generator
	cfg    settings.ChannelConfig
	isLive atomic.Bool
	client *twitch.Client
	db     *database.Database
}

func newChannel(cfg settings.ChannelConfig, channelID int, client *twitch.Client, db *database.Database) *channel {
	return &channel{
		id: channelID,
		markov: markov.New(db, markov.Config{
			ChannelID:        channelID,
			BlacklistedWords: cfg.BlacklistedWords,
			MaxSentenceWords: cfg.MaxSentenceWords,
			AllowNonASCII:    cfg.AllowNonASCIIMessages,
		}),
		cfg:    cfg,
		client: client,
		db:     db,
	}
}

func (c *channel) autoGenerate(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(c.cfg.AutoGenerateInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !c.isLive.Load() {
				slog.Debug("stream offline, skipping auto-generate", "channel", c.cfg.ChannelName)
				continue
			}
			message, err := c.markov.GenerateMessage(ctx)
			if err != nil {
				slog.Error("failed to generate message", "channel", c.cfg.ChannelName, "error", err)
				continue
			}
			c.send(message)
		}
	}
}

func (c *channel) onMessage(ctx context.Context, botUsername string, message twitch.PrivateMessage) {
	c.saveNode(ctx, botUsername, message)

	if strings.EqualFold(message.User.Name, botUsername) {
		return
	}

	if c.cfg.IsUserBlocked(message.User.Name) {
		return
	}

	trimmed := strings.TrimSpace(message.Message)

	if strings.EqualFold(trimmed, "!stats") {
		stats, err := c.markov.GetStatistics(ctx)
		if err != nil {
			slog.Error("failed to get statistics", "channel", c.cfg.ChannelName, "error", err)
			return
		}
		c.client.Reply(c.cfg.ChannelName, message.ID, fmt.Sprintf(
			"Dataset Statistics: Start Pairs: %d, Grammar Entries: %d",
			stats.StartPairs, stats.GrammarEntries,
		))
		return
	}

	if c.cfg.AllowGenerateCommand {
		for _, cmd := range c.cfg.GenerateCommands {
			if strings.HasPrefix(strings.ToLower(trimmed), strings.ToLower(cmd)) {
				if !c.cfg.IsUserAllowed(message.User.Name) {
					slog.Info("generate command denied", "user", message.User.Name, "channel", c.cfg.ChannelName)
					return
				}
				generated, err := c.markov.GenerateMessage(ctx)
				if err != nil {
					slog.Error("failed to generate message", "channel", c.cfg.ChannelName, "error", err)
					return
				}
				if generated != "" {
					c.client.Reply(c.cfg.ChannelName, message.ID, generated)
				}
				return
			}
		}
	}

	if !c.isLive.Load() {
		return
	}

	tokens := tokenizer.Tokenize(message.Message)
	if err := c.markov.TrainMessage(ctx, tokens); err != nil {
		slog.Error("failed to train", "channel", c.cfg.ChannelName, "error", err)
	}
}

func (c *channel) onDelete(ctx context.Context, message twitch.ClearMessage) {
	if err := c.db.DeleteMessageChain(ctx, c.id, message.TargetMsgID, tokenizer.Tokenize); err != nil {
		slog.Error("failed to delete message chain", "channel", c.cfg.ChannelName, "messageID", message.TargetMsgID, "error", err)
		return
	}
	slog.Debug("deleted message chain", "channel", c.cfg.ChannelName, "rootMessageID", message.TargetMsgID)
}

func (c *channel) send(message string) {
	if strings.TrimSpace(message) == "" {
		return
	}
	slog.Info("sending message", "channel", c.cfg.ChannelName, "message", message)
	c.client.Say(c.cfg.ChannelName, message)
}

func (c *channel) saveNode(ctx context.Context, botUsername string, message twitch.PrivateMessage) {
	parentID := ""
	if message.Reply != nil {
		parentID = message.Reply.ParentMsgID
	}
	isBotMessage := strings.EqualFold(message.User.Name, botUsername)
	if err := c.db.SaveMessageChainNode(ctx, c.id, message.ID, parentID, message.Message, isBotMessage); err != nil {
		slog.Error("failed to save message chain node", "channel", c.cfg.ChannelName, "messageID", message.ID, "error", err)
	}
}
