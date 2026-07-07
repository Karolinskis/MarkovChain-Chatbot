package tokenizer

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var emoticonRegex = regexp.MustCompile(`(?i)([<>]?[:;=8][\-o*']?[\)\]\(\[dDpP/:\}\{@|\\]|[\)\]\(\[dDpP/:\}\{@|\\][\-o*']?[:;=8][<>]?|<3)`)

type punctuationRule struct {
	re          *regexp.Regexp
	replacement string
}

var startingQuotes = []punctuationRule{
	{regexp.MustCompile("^([«\"'„]|[`]+)"), " $1 "},
	{regexp.MustCompile("(``)"), " $1 "},
	{regexp.MustCompile(`(?i)(')([^\Wrvlmtsd])`), " $1 $2"},
}

var punctuation = []punctuationRule{
	{regexp.MustCompile(`\x{2019}`), " ’ "},
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
		loc := findEmoticon(sentence)
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

// findEmoticon returns the position of the first emoticon bounded by
// whitespace or string edges, so emoticon-like sequences inside words
// (e.g. the "d;" in "good;") are not treated as emoticons.
func findEmoticon(s string) []int {
	offset := 0
	for {
		loc := emoticonRegex.FindStringIndex(s[offset:])
		if loc == nil {
			return nil
		}

		start, end := offset+loc[0], offset+loc[1]
		if isSpaceBefore(s, start) && isSpaceAfter(s, end) {
			return []int{start, end}
		}
		offset = start + 1
	}
}

func isSpaceBefore(s string, i int) bool {
	if i == 0 {
		return true
	}
	r, _ := utf8.DecodeLastRuneInString(s[:i])
	return unicode.IsSpace(r)
}

func isSpaceAfter(s string, i int) bool {
	if i == len(s) {
		return true
	}
	r, _ := utf8.DecodeRuneInString(s[i:])
	return unicode.IsSpace(r)
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

// prefixTokens attach to the following token when detokenizing.
var prefixTokens = map[string]bool{
	"#":  true,
	`"`:  true,
	"«":  true,
	"„":  true,
	"`":  true,
	"``": true,
}

// Detokenize joins tokens back into a sentence, attaching punctuation.
func Detokenize(tokens []string) string {
	if len(tokens) == 0 {
		return ""
	}

	var result []string
	prefix := false
	for _, token := range tokens {
		switch {
		case prefix:
			result[len(result)-1] += token
		case prefixTokens[token] || len(result) == 0 || !isPunctuation(token):
			result = append(result, token)
		default:
			result[len(result)-1] += token
		}
		prefix = prefixTokens[token]
	}

	return strings.Join(result, " ")
}

func isPunctuation(token string) bool {
	if token == "." {
		return true
	}
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
