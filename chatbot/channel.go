package chatbot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	"markovchain-chatbot/database"
	"markovchain-chatbot/markov"
	"markovchain-chatbot/settings"
	"markovchain-chatbot/tokenizer"

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
		id:     channelID,
		markov: markov.NewGenerator(db, channelID, cfg.BlacklistedWords, cfg.MaxSentenceWords, cfg.AllowNonAsciiMessages),
		cfg:    cfg,
		client: client,
		db:     db,
	}
}

func (c *channel) startAutoGenerate(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Duration(c.cfg.AutoGenerateInterval) * time.Second):
			if !c.isLive.Load() {
				slog.Debug("stream offline, skipping auto-generate", "channel", c.cfg.ChannelName)
				continue
			}
			c.send(c.markov.GenerateMessage(ctx))
		}
	}
}

func (c *channel) onMessage(botUsername string, message twitch.PrivateMessage) {
	ctx := context.Background()

	c.saveNode(ctx, botUsername, message)

	if strings.EqualFold(message.User.Name, botUsername) {
		return
	}

	if c.cfg.IsUserBlocked(message.User.Name) {
		return
	}

	trimmed := strings.TrimSpace(message.Message)

	if strings.EqualFold(trimmed, "!stats") {
		stats := c.markov.GetStatistics(ctx)
		c.client.Reply(c.cfg.ChannelName, message.ID, fmt.Sprintf(
			"Dataset Statistics: Start Pairs: %d, Grammar Entries: %d",
			stats["TotalStartPairs"], stats["TotalGrammarEntries"],
		))
		return
	}

	if c.cfg.AllowGenerateCommand {
		for _, cmd := range c.cfg.GenerateCommands {
			if strings.HasPrefix(strings.ToLower(trimmed), strings.ToLower(cmd)) {
				if c.cfg.IsUserAllowed(message.User.Name) {
					if generated := c.markov.GenerateMessage(ctx); generated != "" {
						c.client.Reply(c.cfg.ChannelName, message.ID, generated)
					}
				} else {
					slog.Info("generate command denied", "user", message.User.Name, "channel", c.cfg.ChannelName)
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

func (c *channel) onDelete(message twitch.ClearMessage) {
	ctx := context.Background()
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
