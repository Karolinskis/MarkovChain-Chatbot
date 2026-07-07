package chatbot

import (
	"context"
	"sync"
	"testing"
	"time"

	"markovchain-chatbot/internal/settings"
)

type fakeIRC struct {
	mu   sync.Mutex
	says []string
}

func (f *fakeIRC) Say(channel, text string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.says = append(f.says, text)
}

func (f *fakeIRC) Reply(channel, messageID, text string) {}

func (f *fakeIRC) said() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.says
}

func TestChannelSend(t *testing.T) {
	irc := &fakeIRC{}
	ch := &channel{cfg: settings.ChannelConfig{ChannelName: "test"}, client: irc}

	ch.send("")
	ch.send("   ")
	if got := irc.said(); len(got) != 0 {
		t.Errorf("send of blank messages should not reach IRC, got %q", got)
	}

	ch.send("hello chat")
	if got := irc.said(); len(got) != 1 || got[0] != "hello chat" {
		t.Errorf("said = %q, want [\"hello chat\"]", got)
	}
}

func TestRunLivePoller(t *testing.T) {
	ch := &channel{cfg: settings.ChannelConfig{ChannelName: "somechannel"}}
	bot := &Bot{channels: map[string]*channel{"somechannel": ch}}

	live := func(channels []string) (map[string]bool, error) {
		statuses := make(map[string]bool, len(channels))
		for _, c := range channels {
			statuses[c] = true
		}
		return statuses, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		RunLivePoller(ctx, live, []*Bot{bot})
		close(done)
	}()

	deadline := time.After(2 * time.Second)
	for !ch.isLive.Load() {
		select {
		case <-deadline:
			t.Fatal("channel never marked live")
		case <-time.After(10 * time.Millisecond):
		}
	}

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("RunLivePoller did not stop on context cancellation")
	}
}

func TestRunLivePollerNoChannels(t *testing.T) {
	live := func(channels []string) (map[string]bool, error) {
		t.Error("LiveChecker should not be called with no channels")
		return nil, nil
	}

	done := make(chan struct{})
	go func() {
		RunLivePoller(context.Background(), live, nil)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("RunLivePoller with no channels should return immediately")
	}
}
