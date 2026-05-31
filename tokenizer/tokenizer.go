package tokenizer

import (
	"regexp"
	"strings"
)

var emoticonRegex = regexp.MustCompile(`(?i)([<>]?[:;=8][\-o*']?[\)\]\(\[dDpP/:\}\{@|\\]|[\)\]\(\[dDpP/:\}\{@|\\][\-o*']?[:;=8][<>]?|<3)`)

type punctuationRule struct {
	re          *regexp.Regexp
	replacement string
}

var startingQuotes = []punctuationRule{
	{regexp.MustCompile("([«\"'„]|[`]+)"), " $1 "},
	{regexp.MustCompile("(``)"), " $1 "},
	{regexp.MustCompile(`(?i)(')([^\Wrvlmtsd])`), " $1 $2"},
}

var punctuation = []punctuationRule{
	{regexp.MustCompile(`\x{2019}`), " \u2019 "},
	{regexp.MustCompile(`([^\.])(\.)([\]\)}>»\x{201D}\x{2019}]*)\s*$`), "$1 $2$3 "},
	{regexp.MustCompile(`([:,])([^\d])`), " $1 $2"},
	{regexp.MustCompile(`([:,])$`), " $1 "},
	{regexp.MustCompile(`\.{2,}`), " $0 "},
	{regexp.MustCompile(`[;#$%&]`), " $0 "},
	{regexp.MustCompile(`([^\.])(\.)([\]\)}>\"']*)\s*$`), "$1 $2$3 "},
	{regexp.MustCompile(`[?!]`), " $0 "},
	{regexp.MustCompile(`([^'])'` + " "), "$1 ' "},
	{regexp.MustCompile(`[*]`), " $0 "},
}

// Tokenize splits a sentence into tokens, preserving emoticons.
func Tokenize(sentence string) []string {
	var output []string

	for {
		loc := emoticonRegex.FindStringIndex(sentence)
		if loc == nil {
			break
		}

		before := strings.TrimSpace(sentence[:loc[0]])
		emoticon := sentence[loc[0]:loc[1]]
		sentence = strings.TrimSpace(sentence[loc[1]:])

		output = append(output, tokenizePart(before)...)
		output = append(output, emoticon)
	}

	output = append(output, tokenizePart(sentence)...)
	return output
}

func tokenizePart(sentence string) []string {
	for _, rule := range startingQuotes {
		sentence = rule.re.ReplaceAllString(sentence, rule.replacement)
	}

	for _, rule := range punctuation {
		sentence = rule.re.ReplaceAllString(sentence, rule.replacement)
	}

	var tokens []string
	for _, t := range strings.Fields(sentence) {
		if t != "" {
			tokens = append(tokens, t)
		}
	}
	return tokens
}

// Detokenize joins tokens back into a sentence, attaching punctuation.
func Detokenize(tokens []string) string {
	if len(tokens) == 0 {
		return ""
	}

	var result []string
	for i, token := range tokens {
		if i > 0 && isPunctuation(token) {
			result[len(result)-1] += token
		} else {
			result = append(result, token)
		}
	}

	return strings.Join(result, " ")
}

func isPunctuation(token string) bool {
	if emoticonRegex.MatchString(token) {
		return false
	}
	for _, rule := range punctuation {
		if rule.re.MatchString(token) {
			return true
		}
	}
	return false
}
