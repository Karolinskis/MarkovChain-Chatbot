package chatbot

import (
	"context"
	"strings"
	"testing"
	"time"

	"markovchain-chatbot/internal/database"
	"markovchain-chatbot/internal/llm"
	"markovchain-chatbot/internal/markov"
	"markovchain-chatbot/internal/settings"

	twitch "github.com/gempir/go-twitch-irc/v4"
)

type fakeChatter struct {
	reply string
	got   []llm.Message
}

func (f *fakeChatter) Chat(ctx context.Context, messages []llm.Message) (string, error) {
	f.got = messages
	return f.reply, nil
}

func TestAddressesBot(t *testing.T) {
	tests := []struct {
		name    string
		message twitch.PrivateMessage
		want    bool
	}{
		{"mention", twitch.PrivateMessage{Message: "hey @MyBot how are you"}, true},
		{"mention with punctuation", twitch.PrivateMessage{Message: "@mybot, hi"}, true},
		{"reply to bot", twitch.PrivateMessage{Message: "lol", Reply: &twitch.Reply{ParentUserLogin: "mybot"}}, true},
		{"reply to someone else", twitch.PrivateMessage{Message: "@alice lol", Reply: &twitch.Reply{ParentUserLogin: "alice"}}, false},
		{"name without @", twitch.PrivateMessage{Message: "mybot is funny"}, false},
		{"longer name", twitch.PrivateMessage{Message: "@mybot2 hi"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := addressesBot(tt.message, "mybot"); got != tt.want {
				t.Errorf("addressesBot(%q) = %v, want %v", tt.message.Message, got, tt.want)
			}
		})
	}
}

func TestCleanLLMReply(t *testing.T) {
	tests := []struct {
		name  string
		reply string
		want  string
	}{
		{"plain", "sounds good", "sounds good"},
		{"multi-line flattened", "first line\n\nsecond  line", "first line second line"},
		{"name prefix dropped", "MyBot: sure thing", "sure thing"},
		{"quotes trimmed", `"quoted reply"`, "quoted reply"},
		{"capped at limit", strings.Repeat("a", 600), strings.Repeat("a", twitchMaxMessage)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cleanLLMReply(tt.reply, "mybot"); got != tt.want {
				t.Errorf("cleanLLMReply(%q) = %q, want %q", tt.reply, got, tt.want)
			}
		})
	}
}

func TestAllowLLMReply(t *testing.T) {
	ch := &channel{cfg: settings.ChannelConfig{LLMCooldown: 30}, llmLastReply: make(map[string]time.Time)}
	now := time.Now()

	if !ch.allowLLMReply("alice", now) {
		t.Fatal("first reply should be allowed")
	}
	if ch.allowLLMReply("Alice", now.Add(10*time.Second)) {
		t.Error("reply within cooldown should be denied, case-insensitively")
	}
	if !ch.allowLLMReply("bob", now.Add(10*time.Second)) {
		t.Error("cooldown should be per user")
	}
	if !ch.allowLLMReply("alice", now.Add(31*time.Second)) {
		t.Error("reply after cooldown should be allowed")
	}
}

func TestBuildLLMMessages(t *testing.T) {
	ch := &channel{cfg: settings.ChannelConfig{ChannelName: "somechannel", LLMPersona: "You love speedruns."}}
	thread := []database.ThreadMessage{
		{SenderUsername: "mybot", Text: "that boss was easy", IsBotMessage: true},
		{SenderUsername: "alice", Text: "@mybot no way"},
	}

	got := ch.buildLLMMessages("mybot", thread)
	if len(got) != 3 {
		t.Fatalf("got %d messages, want 3: %+v", len(got), got)
	}
	if got[0].Role != "system" || !strings.Contains(got[0].Content, "somechannel") || !strings.HasSuffix(got[0].Content, "You love speedruns.") {
		t.Errorf("system = %+v", got[0])
	}
	if got[1] != (llm.Message{Role: "assistant", Content: "that boss was easy"}) {
		t.Errorf("bot message = %+v", got[1])
	}
	if got[2] != (llm.Message{Role: "user", Content: "alice: no way"}) {
		t.Errorf("user message = %+v", got[2])
	}
}

func TestReplyWithLLM(t *testing.T) {
	tests := []struct {
		name    string
		reply   string
		replies []string
	}{
		{"clean reply sent", "pretty good stream", []string{"pretty good stream"}},
		{"link filtered", "check out example.com", nil},
		{"blacklisted word filtered", "that is badword", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			irc := &fakeIRC{}
			fake := &fakeChatter{reply: tt.reply}
			ch := &channel{
				cfg:    settings.ChannelConfig{ChannelName: "somechannel"},
				client: irc,
				llm:    fake,
				markov: markov.New(nil, markov.Config{BlacklistedWords: []string{"badword"}}),
			}

			ch.replyWithLLM(context.Background(), "mybot", twitch.PrivateMessage{
				ID:      "msg-1",
				User:    twitch.User{Name: "alice"},
				Message: "@mybot how is the stream",
			})

			irc.mu.Lock()
			defer irc.mu.Unlock()
			if len(irc.replies) != len(tt.replies) || (len(tt.replies) > 0 && irc.replies[0] != tt.replies[0]) {
				t.Errorf("replies = %q, want %q", irc.replies, tt.replies)
			}
			if last := fake.got[len(fake.got)-1]; last.Content != "alice: how is the stream" {
				t.Errorf("last prompt message = %+v", last)
			}
		})
	}
}
