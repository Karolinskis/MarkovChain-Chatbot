package markov

import (
	"context"
	"log/slog"
	"strings"

	"markovchain-chatbot/internal/database"
	"markovchain-chatbot/internal/filter"
	"markovchain-chatbot/internal/tokenizer"
)

const maxGenerationAttempts = 10

// Config holds the per-channel generation settings for a Generator.
type Config struct {
	ChannelID        int
	BlacklistedWords []string
	MaxSentenceWords int
	AllowNonASCII    bool
}

type Generator struct {
	db                         *database.Database
	channelID                  int
	normalizedBlacklistedWords []string
	maxSentenceWords           int
	allowNonASCII              bool
}

func New(db *database.Database, cfg Config) *Generator {
	normalized := make([]string, 0, len(cfg.BlacklistedWords))
	for _, w := range cfg.BlacklistedWords {
		normalized = append(normalized, filter.Normalize(w))
	}

	return &Generator{
		db:                         db,
		channelID:                  cfg.ChannelID,
		normalizedBlacklistedWords: normalized,
		maxSentenceWords:           cfg.MaxSentenceWords,
		allowNonASCII:              cfg.AllowNonASCII,
	}
}

func (g *Generator) TrainMessage(ctx context.Context, tokens []string) error {
	if len(tokens) < 2 {
		return nil
	}

	if err := g.db.AddStart(ctx, g.channelID, tokens[0], tokens[1]); err != nil {
		return err
	}

	for i := 0; i < len(tokens)-2; i++ {
		w3 := tokens[i+2]
		if err := g.db.AddGrammar(ctx, g.channelID, tokens[i], tokens[i+1], &w3); err != nil {
			return err
		}
	}

	return g.db.AddGrammar(ctx, g.channelID, tokens[len(tokens)-2], tokens[len(tokens)-1], nil)
}

// GenerateMessage builds a random sentence from the trained dataset. It
// returns "" without error when no clean sentence could be produced within
// the attempt limit.
func (g *Generator) GenerateMessage(ctx context.Context) (string, error) {
	for i := 0; i < maxGenerationAttempts; i++ {
		startWordPair, err := g.db.GetStartWord(ctx, g.channelID)
		if err != nil {
			return "", err
		}
		if startWordPair == "" {
			continue
		}

		sentence, err := g.tryGenerateSentence(ctx, startWordPair)
		if err != nil {
			return "", err
		}
		if len(sentence) == 0 {
			continue
		}

		message := tokenizer.Detokenize(sentence)
		if filter.IsCleanMessage(message, g.allowNonASCII) {
			return message, nil
		}
	}

	slog.Warn("failed to generate clean sentence", "attempts", maxGenerationAttempts)
	return "", nil
}

func (g *Generator) GetStatistics(ctx context.Context) (database.Statistics, error) {
	return g.db.GetStatistics(ctx, g.channelID)
}

func (g *Generator) tryGenerateSentence(ctx context.Context, startWordPair string) ([]string, error) {
	words := strings.SplitN(startWordPair, " ", 2)
	if len(words) < 2 || g.areWordsBlacklisted(words) {
		return nil, nil
	}

	result := make([]string, 0, g.maxSentenceWords)
	result = append(result, words...)
	currentWord1 := words[0]
	currentWord2 := words[1]

	for i := 0; i < g.maxSentenceWords-2; i++ {
		nextWord, err := g.db.GetNextWord(ctx, g.channelID, currentWord1, currentWord2)
		if err != nil {
			return nil, err
		}
		if nextWord == "" {
			break
		}

		if g.isWordBlacklisted(nextWord) {
			slog.Debug("blacklisted word hit", "partial", strings.Join(result, " "), "word", nextWord)
			return nil, nil
		}

		result = append(result, nextWord)
		currentWord1 = currentWord2
		currentWord2 = nextWord
	}

	return result, nil
}

func (g *Generator) isWordBlacklisted(word string) bool {
	if len(g.normalizedBlacklistedWords) == 0 {
		return false
	}
	normalizedWord := filter.Normalize(word)
	for _, bw := range g.normalizedBlacklistedWords {
		if bw == normalizedWord {
			return true
		}
	}
	return false
}

func (g *Generator) areWordsBlacklisted(words []string) bool {
	for _, w := range words {
		if g.isWordBlacklisted(w) {
			return true
		}
	}
	return false
}
