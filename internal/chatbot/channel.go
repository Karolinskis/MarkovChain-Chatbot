package chatbot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"markovchain-chatbot/internal/database"
	"markovchain-chatbot/internal/llm"
	"markovchain-chatbot/internal/markov"
	"markovchain-chatbot/internal/metrics"
	"markovchain-chatbot/internal/settings"
	"markovchain-chatbot/internal/tokenizer"

	twitch "github.com/gempir/go-twitch-irc/v4"
)

// ircClient is the part of the Twitch IRC client a channel sends through.
type ircClient interface {
	Say(channel, text string)
	Reply(channel, messageID, text string)
}

type channel struct {
	id     int
	markov *markov.Generator
	cfg    settings.ChannelConfig
	isLive atomic.Bool
	client ircClient
	db     *database.Database

	llm          chatter
	llmBusy      atomic.Bool
	llmMu        sync.Mutex
	llmLastReply map[string]time.Time
}

func newChannel(cfg settings.ChannelConfig, channelID int, client ircClient, db *database.Database, llmClient *llm.Client) *channel {
	ch := &channel{
		id: channelID,
		markov: markov.New(db, markov.Config{
			ChannelID:        channelID,
			BlacklistedWords: cfg.BlacklistedWords,
			MaxSentenceWords: cfg.MaxSentenceWords,
			AllowNonASCII:    cfg.AllowNonASCIIMessages,
		}),
		cfg:          cfg,
		client:       client,
		db:           db,
		llmLastReply: make(map[string]time.Time),
	}
	if llmClient != nil {
		ch.llm = llmClient
	}
	return ch
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
				metrics.GenerationErrors.WithLabelValues(c.cfg.ChannelName, "auto").Inc()
				continue
			}
			if message == "" {
				metrics.GenerationEmpty.WithLabelValues(c.cfg.ChannelName, "auto").Inc()
				continue
			}
			metrics.MessagesGenerated.WithLabelValues(c.cfg.ChannelName, "auto").Inc()
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
		if args, ok := matchCommand(trimmed, c.cfg.GenerateCommands); ok {
			if !c.cfg.IsUserAllowed(message.User.Name) {
				slog.Info("generate command denied", "user", message.User.Name, "channel", c.cfg.ChannelName)
				return
			}
			// "!command some question" is answered by the LLM when enabled.
			if args != "" && c.llm != nil && c.cfg.LLMReplies {
				if c.allowLLMReply(message.User.Name, time.Now()) {
					message.Message = args
					go c.replyWithLLM(ctx, botUsername, message)
				}
				return
			}
			generated, err := c.markov.GenerateMessage(ctx)
			if err != nil {
				slog.Error("failed to generate message", "channel", c.cfg.ChannelName, "error", err)
				metrics.GenerationErrors.WithLabelValues(c.cfg.ChannelName, "command").Inc()
				return
			}
			if generated == "" {
				metrics.GenerationEmpty.WithLabelValues(c.cfg.ChannelName, "command").Inc()
				return
			}
			metrics.MessagesGenerated.WithLabelValues(c.cfg.ChannelName, "command").Inc()
			c.client.Reply(c.cfg.ChannelName, message.ID, generated)
			return
		}
	}

	if c.llm != nil && c.cfg.LLMReplies && addressesBot(message, botUsername) && c.allowLLMReply(message.User.Name, time.Now()) {
		go c.replyWithLLM(ctx, botUsername, message)
	}

	if !c.isLive.Load() {
		return
	}

	tokens := tokenizer.Tokenize(message.Message)
	if err := c.markov.TrainMessage(ctx, tokens); err != nil {
		slog.Error("failed to train", "channel", c.cfg.ChannelName, "error", err)
		metrics.TrainErrors.WithLabelValues(c.cfg.ChannelName).Inc()
		return
	}
	metrics.MessagesTrained.WithLabelValues(c.cfg.ChannelName).Inc()
}

func (c *channel) onDelete(ctx context.Context, message twitch.ClearMessage) {
	if err := c.db.DeleteMessageChain(ctx, c.id, message.TargetMsgID, tokenizer.Tokenize); err != nil {
		slog.Error("failed to delete message chain", "channel", c.cfg.ChannelName, "messageID", message.TargetMsgID, "error", err)
		metrics.UntrainErrors.WithLabelValues(c.cfg.ChannelName).Inc()
		return
	}
	metrics.MessagesUntrained.WithLabelValues(c.cfg.ChannelName).Inc()
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
	if err := c.db.SaveMessageChainNode(ctx, c.id, message.ID, parentID, message.Message, message.User.Name, isBotMessage); err != nil {
		slog.Error("failed to save message chain node", "channel", c.cfg.ChannelName, "messageID", message.ID, "error", err)
	}
}

// matchCommand reports whether text starts with one of commands as a whole
// word, and returns the text after it.
func matchCommand(text string, commands []string) (args string, ok bool) {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return "", false
	}
	for _, cmd := range commands {
		if strings.EqualFold(fields[0], cmd) {
			return strings.TrimSpace(strings.TrimSpace(text)[len(fields[0]):]), true
		}
	}
	return "", false
}
