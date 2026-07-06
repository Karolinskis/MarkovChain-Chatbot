package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

type Settings struct {
	DatabaseURL           string   `json:"DatabaseURL"`
	BotUsername           string   `json:"BotUsername"`
	AccessToken           string   `json:"AccessToken"`
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
	EnableDiscordLogging  bool     `json:"EnableDiscordLogging"`
	DiscordWebhookURL     string   `json:"DiscordWebhookUrl"`
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
		DatabaseURL:           "postgres://user:password@localhost:5432/markovbot",
		BotUsername:           "botUsername",
		AccessToken:           "accessToken",
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
		EnableDiscordLogging:  false,
		DiscordWebhookURL:     "",
		AllowNonAsciiMessages: false,
	}

	data, err := json.MarshalIndent(&cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (s *Settings) IsUserBlocked(username string) bool {
	for _, u := range s.BlockedUsers {
		if strings.EqualFold(u, username) {
			return true
		}
	}
	return false
}

func (s *Settings) IsUserAllowed(username string) bool {
	for _, u := range s.AllowedUsers {
		if u == "*" || strings.EqualFold(u, username) {
			return true
		}
	}
	return false
}
