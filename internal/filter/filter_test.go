package filter

import "testing"

func TestIsCleanMessage(t *testing.T) {
	tests := []struct {
		name          string
		message       string
		allowNonASCII bool
		want          bool
	}{
		{name: "plain text", message: "hello world", want: true},
		{name: "http link", message: "check http://example.com out", want: false},
		{name: "www link", message: "www.example.com stuff", want: false},
		{name: "bare domain", message: "visit example.com", want: false},
		{name: "mention", message: "@someone hi", want: false},
		{name: "command", message: "!command arg", want: false},
		{name: "dot command", message: ".command arg", want: false},
		{name: "exclamation mid-sentence", message: "wow! nice", want: true},
		{name: "diacritics normalized to ascii", message: "café time", want: true},
		{name: "non-ascii blocked", message: "日本語", want: false},
		{name: "non-ascii allowed", message: "日本語", allowNonASCII: true, want: true},
		{name: "emoji blocked", message: "nice 🙂", want: false},
		{name: "emoji allowed", message: "nice 🙂", allowNonASCII: true, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsCleanMessage(tt.message, tt.allowNonASCII); got != tt.want {
				t.Errorf("IsCleanMessage(%q, %v) = %v, want %v", tt.message, tt.allowNonASCII, got, tt.want)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"café", "cafe"},
		{"ÀÉÎÕÜ", "AEIOU"},
		{"žąsis", "zasis"},
		{"hello", "hello"},
		{"!?.,", "!?.,"},
		{"", ""},
	}

	for _, tt := range tests {
		if got := Normalize(tt.in); got != tt.want {
			t.Errorf("Normalize(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
