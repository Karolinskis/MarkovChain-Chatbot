package filter

import (
	"log/slog"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var (
	linkRegex    = regexp.MustCompile(`(http[^\s]+|www\.[^\s]+|[^\s]+\.[a-z]{2,})`)
	mentionRegex = regexp.MustCompile(`@\w+`)
	commandRegex = regexp.MustCompile(`^[!.,]\w+`)
)

// IsCleanMessage checks if the message is free of links, mentions, and commands.
// allowNonASCII controls whether non-ASCII characters are permitted.
func IsCleanMessage(message string, allowNonASCII bool) bool {
	normalized := Normalize(message)

	if !allowNonASCII {
		for _, c := range normalized {
			if c > 127 {
				slog.Debug("blocked message", "message", message, "reason", "non-ASCII characters")
				return false
			}
		}
	}

	if linkRegex.MatchString(normalized) {
		slog.Debug("blocked message", "message", message, "reason", "contains link")
		return false
	}

	if mentionRegex.MatchString(normalized) {
		slog.Debug("blocked message", "message", message, "reason", "contains mention")
		return false
	}

	if commandRegex.MatchString(normalized) {
		slog.Debug("blocked message", "message", message, "reason", "contains command")
		return false
	}

	return true
}

// Normalize replaces diacritical marks with their base characters.
func Normalize(input string) string {
	var result strings.Builder
	for _, c := range input {
		if unicode.IsSymbol(c) || unicode.IsPunct(c) {
			result.WriteRune(c)
		} else {
			result.WriteRune(NormalizeChar(c))
		}
	}
	return result.String()
}

// NormalizeChar removes diacritical marks from a character.
func NormalizeChar(c rune) rune {
	decomposed := norm.NFD.String(string(c))
	for _, r := range decomposed {
		if !unicode.Is(unicode.Mn, r) {
			return r
		}
	}
	return c
}
