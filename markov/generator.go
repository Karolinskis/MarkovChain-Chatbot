package markov

import (
	"log/slog"
	"strings"

	"markovchain-chatbot/database"
	"markovchain-chatbot/filter"
	"markovchain-chatbot/tokenizer"
)

const maxGenerationAttempts = 10

type Generator struct {
	db                         *database.Database
	normalizedBlacklistedWords []string
	maxSentenceWords           int
}

func NewGenerator(db *database.Database, blacklistedWords []string, maxSentenceWords int) *Generator {
	normalized := make([]string, 0, len(blacklistedWords))
	for _, w := range blacklistedWords {
		normalized = append(normalized, filter.Normalize(w))
	}

	return &Generator{
		db:                         db,
		normalizedBlacklistedWords: normalized,
		maxSentenceWords:           maxSentenceWords,
	}
}

func (g *Generator) Train(tokens []string) error {
	if len(tokens) < 2 {
		return nil
	}

	if err := g.db.AddStart(tokens[0], tokens[1]); err != nil {
		return err
	}

	for i := 0; i < len(tokens)-2; i++ {
		if err := g.db.AddGrammar(tokens[i], tokens[i+1], tokens[i+2]); err != nil {
			return err
		}
	}

	return g.db.AddGrammar(tokens[len(tokens)-2], tokens[len(tokens)-1], "<END>")
}

func (g *Generator) GenerateMessage() string {
	for i := 0; i < maxGenerationAttempts; i++ {
		startWordPair := g.db.GetStartWord()
		if startWordPair == "" {
			continue
		}

		sentence := g.tryGenerateSentence(startWordPair)
		if len(sentence) > 0 {
			return tokenizer.Detokenize(sentence)
		}
	}

	slog.Warn("failed to generate clean sentence", "attempts", maxGenerationAttempts)
	return ""
}

func (g *Generator) GetStatistics() map[string]int {
	return g.db.GetStatistics()
}

func (g *Generator) tryGenerateSentence(startWordPair string) []string {
	words := strings.SplitN(startWordPair, " ", 2)
	if len(words) < 2 || g.areWordsBlacklisted(words) {
		return nil
	}

	result := make([]string, 0, g.maxSentenceWords)
	result = append(result, words...)
	currentWord1 := words[0]
	currentWord2 := words[1]

	for i := 0; i < g.maxSentenceWords-2; i++ {
		nextWord := g.db.GetNextWord(currentWord1, currentWord2)
		if nextWord == "" || nextWord == "<END>" {
			break
		}

		if g.isWordBlacklisted(nextWord) {
			slog.Debug("blacklisted word hit", "partial", strings.Join(result, " "), "word", nextWord)
			return nil
		}

		result = append(result, nextWord)
		currentWord1 = currentWord2
		currentWord2 = nextWord
	}

	return result
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
