package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		if _, err := Load(filepath.Join(t.TempDir(), "settings.json")); err == nil {
			t.Fatal("Load() on missing file should return an error")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "settings.json")
		if err := os.WriteFile(path, []byte("{invalid"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Fatal("Load() on invalid JSON should return an error")
		}
	})

	t.Run("valid", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "settings.json")
		data := `{
			"DatabaseURL": "postgres://localhost/test",
			"Bots": [{
				"BotUsername": "mybot",
				"AccessToken": "token",
				"Channels": [{"ChannelName": "mychannel", "MaxSentenceWords": 25}]
			}]
		}`
		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}

		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("Load() = %v", err)
		}
		if cfg.DatabaseURL != "postgres://localhost/test" {
			t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
		}
		if len(cfg.Bots) != 1 || cfg.Bots[0].BotUsername != "mybot" {
			t.Fatalf("Bots = %+v", cfg.Bots)
		}
		ch := cfg.Bots[0].Channels[0]
		if ch.ChannelName != "mychannel" || ch.MaxSentenceWords != 25 {
			t.Errorf("Channels[0] = %+v", ch)
		}
	})
}

func TestIsUserAllowed(t *testing.T) {
	tests := []struct {
		name    string
		allowed []string
		user    string
		want    bool
	}{
		{"wildcard", []string{"*"}, "anyone", true},
		{"exact match", []string{"alice"}, "alice", true},
		{"case insensitive", []string{"Alice"}, "aLiCe", true},
		{"not listed", []string{"alice"}, "bob", false},
		{"empty list", nil, "alice", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ChannelConfig{AllowedUsers: tt.allowed}
			if got := cfg.IsUserAllowed(tt.user); got != tt.want {
				t.Errorf("IsUserAllowed(%q) with %q = %v, want %v", tt.user, tt.allowed, got, tt.want)
			}
		})
	}
}

func TestIsUserBlocked(t *testing.T) {
	tests := []struct {
		name    string
		blocked []string
		user    string
		want    bool
	}{
		{"exact match", []string{"nightbot"}, "nightbot", true},
		{"case insensitive", []string{"NightBot"}, "nightbot", true},
		{"not listed", []string{"nightbot"}, "alice", false},
		{"empty list", nil, "alice", false},
		{"no wildcard support", []string{"*"}, "alice", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ChannelConfig{BlockedUsers: tt.blocked}
			if got := cfg.IsUserBlocked(tt.user); got != tt.want {
				t.Errorf("IsUserBlocked(%q) with %q = %v, want %v", tt.user, tt.blocked, got, tt.want)
			}
		})
	}
}
