package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

type Settings struct {
	DatabaseURL       string      `json:"DatabaseURL"`
	HelixClientID     string      `json:"HelixClientID"`
	HelixClientSecret string      `json:"HelixClientSecret"`
	Bots              []BotConfig `json:"Bots"`
}

type BotConfig struct {
	BotUsername string          `json:"BotUsername"`
	AccessToken string          `json:"AccessToken"`
	Channels    []ChannelConfig `json:"Channels"`
}

type ChannelConfig struct {
	ChannelName           string   `json:"ChannelName"`
	TrainingMode          bool     `json:"TrainingMode"`
	AllowedUsers          []string `json:"AllowedUsers"`
	BlockedUsers          []string `json:"BlockedUsers"`
	MinSentenceWords      int      `json:"MinSentenceWords"`
	MaxSentenceWords      int      `json:"MaxSentenceWords"`
	AutoGenerateMessages  bool     `json:"AutoGenerateMessages"`
	AutoGenerateInterval  int      `json:"AutoGenerateInterval"`
	AllowGenerateCommand  bool     `json:"AllowGenerateCommand"`
	GenerateCommands      []string `json:"GenerateCommands"`
	BlacklistedWords      []string `json:"BlacklistedWords"`
	AllowNonAsciiMessages bool     `json:"AllowNonAsciiMessages"`
}

// Load reads settings from the given path. If the file doesn't exist,
// it generates a default config and returns an error.
func Load(path string) (*Settings, error) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := writeDefaults(path); err != nil {
			return nil, fmt.Errorf("generating default settings: %w", err)
		}
		return nil, fmt.Errorf("settings file not found, generated defaults at %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading settings file: %w", err)
	}

	var cfg Settings
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing settings: %w", err)
	}

	return &cfg, nil
}

func writeDefaults(path string) error {
	cfg := Settings{
		DatabaseURL:       "postgres://user:password@localhost:5432/markovbot",
		HelixClientID:     "your_twitch_app_client_id",
		HelixClientSecret: "your_twitch_app_client_secret",
		Bots: []BotConfig{
			{
				BotUsername: "botUsername",
				AccessToken: "accessToken",
				Channels: []ChannelConfig{
					{
						ChannelName:           "channelName",
						TrainingMode:          false,
						AllowedUsers:          []string{"allowedUser1", "allowedUser2"},
						BlockedUsers:          []string{"blockedUser1", "blockedUser2"},
						MinSentenceWords:      -1,
						MaxSentenceWords:      20,
						AutoGenerateMessages:  true,
						AutoGenerateInterval:  5000,
						AllowGenerateCommand:  true,
						GenerateCommands:      []string{"!generate"},
						BlacklistedWords:      []string{},
						AllowNonAsciiMessages: false,
					},
				},
			},
		},
	}

	data, err := json.MarshalIndent(&cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (c *ChannelConfig) IsUserBlocked(username string) bool {
	for _, u := range c.BlockedUsers {
		if strings.EqualFold(u, username) {
			return true
		}
	}
	return false
}

func (c *ChannelConfig) IsUserAllowed(username string) bool {
	for _, u := range c.AllowedUsers {
		if u == "*" || strings.EqualFold(u, username) {
			return true
		}
	}
	return false
}
