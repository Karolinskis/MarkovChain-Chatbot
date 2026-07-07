package tokenizer

import (
	"slices"
	"strings"
	"testing"
)

var cases = []struct {
	name     string
	sentence string
	tokens   []string
}{
	{name: "plain words", sentence: "hello world", tokens: []string{"hello", "world"}},
	{name: "comma and exclamation", sentence: "hello, world!", tokens: []string{"hello", ",", "world", "!"}},
	{name: "emoticon preserved", sentence: "KEKW :D nice", tokens: []string{"KEKW", ":D", "nice"}},
	{name: "heart emoticon", sentence: "<3 you", tokens: []string{"<3", "you"}},
	{name: "contraction kept whole", sentence: "I'm happy :)", tokens: []string{"I'm", "happy", ":)"}},
	{name: "emoticon-like sequence inside word", sentence: "good; fine", tokens: []string{"good", ";", "fine"}},
	{name: "stacked terminators", sentence: "what?!", tokens: []string{"what", "?", "!"}},
	{name: "ellipsis kept whole", sentence: "wait... really", tokens: []string{"wait", "...", "really"}},
	{name: "trailing period", sentence: "end of sentence.", tokens: []string{"end", "of", "sentence", "."}},
	{name: "hash attaches forward", sentence: "50% off #deal", tokens: []string{"50", "%", "off", "#", "deal"}},
	{name: "whitespace collapsed", sentence: "multiple   spaces here", tokens: []string{"multiple", "spaces", "here"}},
	{name: "empty input", sentence: "", tokens: nil},
	{name: "single word", sentence: "one", tokens: []string{"one"}},
	{name: "number with comma kept whole", sentence: "5,000 dollars", tokens: []string{"5,000", "dollars"}},
	{name: "colon split before non-digit", sentence: "time: 5pm", tokens: []string{"time", ":", "5pm"}},
	{name: "leading quote reattached", sentence: `"quoted text`, tokens: []string{`"`, "quoted", "text"}},
	{name: "leading punctuation stays separate", sentence: "! hello", tokens: []string{"!", "hello"}},
	{name: "trailing hash stays separate", sentence: "such #", tokens: []string{"such", "#"}},
}

func TestTokenize(t *testing.T) {
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := Tokenize(tt.sentence); !slices.Equal(got, tt.tokens) {
				t.Errorf("Tokenize(%q) = %q, want %q", tt.sentence, got, tt.tokens)
			}
		})
	}
}

func TestDetokenize(t *testing.T) {
	for _, tt := range cases {
		// Detokenize restores the sentence up to whitespace normalization.
		want := strings.Join(strings.Fields(tt.sentence), " ")
		t.Run(tt.name, func(t *testing.T) {
			if got := Detokenize(tt.tokens); got != want {
				t.Errorf("Detokenize(%q) = %q, want %q", tt.tokens, got, want)
			}
		})
	}
}
