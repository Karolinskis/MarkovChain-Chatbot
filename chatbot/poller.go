package chatbot

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

const streamPollInterval = 60 * time.Second

// StartLivePoller periodically checks the live status of every channel across
// the given bots with a single LiveChecker call per tick and updates each
// channel's live flag. It blocks until ctx is cancelled.
func StartLivePoller(ctx context.Context, live LiveChecker, bots []*Bot) {
	type target struct {
		login string
		ch    *channel
	}
	var targets []target
	var logins []string
	seen := make(map[string]bool)
	for _, bot := range bots {
		for name, ch := range bot.channels {
			login := strings.ToLower(name)
			targets = append(targets, target{login, ch})
			if !seen[login] {
				seen[login] = true
				logins = append(logins, login)
			}
		}
	}
	if len(targets) == 0 {
		return
	}

	check := func() {
		statuses, err := live(logins)
		if err != nil {
			slog.Warn("stream status check failed", "error", err)
			return
		}
		for _, t := range targets {
			nowLive := statuses[t.login]
			if wasLive := t.ch.isLive.Swap(nowLive); nowLive != wasLive {
				if nowLive {
					slog.Info("stream live", "channel", t.login)
				} else {
					slog.Info("stream offline", "channel", t.login)
				}
			}
		}
	}

	check()

	ticker := time.NewTicker(streamPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			check()
		}
	}
}
