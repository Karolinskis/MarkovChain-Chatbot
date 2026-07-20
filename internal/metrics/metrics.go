// Package metrics exposes Prometheus counters and gauges for the bot's
// training, generation and connection pipelines.
package metrics

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// MessagesTrained counts chat messages folded into a channel's Markov
	// chain.
	MessagesTrained = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "markovbot_messages_trained_total",
		Help: "Chat messages trained into the Markov chain, by channel.",
	}, []string{"channel"})

	TrainErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "markovbot_train_errors_total",
		Help: "Messages that failed to train, by channel.",
	}, []string{"channel"})

	// MessagesGenerated counts messages sent to chat, by channel and
	// trigger ("auto" or "command").
	MessagesGenerated = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "markovbot_messages_generated_total",
		Help: "Messages generated and sent to chat, by channel and trigger.",
	}, []string{"channel", "trigger"})

	GenerationErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "markovbot_generation_errors_total",
		Help: "Generation attempts that returned an error, by channel and trigger.",
	}, []string{"channel", "trigger"})

	// GenerationEmpty counts generation attempts that produced no clean
	// sentence within the attempt limit (thin dataset, blacklist hits).
	GenerationEmpty = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "markovbot_generation_empty_total",
		Help: "Generation attempts that produced no clean sentence, by channel and trigger.",
	}, []string{"channel", "trigger"})

	// MessagesUntrained counts message chains removed after a moderator
	// deletion.
	MessagesUntrained = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "markovbot_messages_untrained_total",
		Help: "Message chains untrained after a moderator deletion, by channel.",
	}, []string{"channel"})

	UntrainErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "markovbot_untrain_errors_total",
		Help: "Untrain attempts that failed, by channel.",
	}, []string{"channel"})

	// StreamLive reports live status per channel: 1 live, 0 offline.
	StreamLive = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "markovbot_stream_live",
		Help: "1 while the channel's stream is live, 0 otherwise.",
	}, []string{"channel"})

	LiveCheckErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "markovbot_live_check_errors_total",
		Help: "Failed Twitch Helix live-status checks.",
	})

	// IRCUp reports whether a bot account's Twitch IRC connection is up.
	IRCUp = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "markovbot_irc_up",
		Help: "1 while the bot's Twitch IRC connection is active.",
	}, []string{"bot"})
)

// Serve exposes /metrics on addr in a background goroutine.
func Serve(addr string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			slog.Error("metrics server", "err", err)
		}
	}()
}
