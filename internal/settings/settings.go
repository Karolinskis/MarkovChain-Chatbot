package settings

import (
	"encoding/json"
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
	MaxSentenceWords      int      `json:"MaxSentenceWords"`
	AutoGenerateMessages  bool     `json:"AutoGenerateMessages"`
	AutoGenerateInterval  int      `json:"AutoGenerateInterval"`
	AllowGenerateCommand  bool     `json:"AllowGenerateCommand"`
	GenerateCommands      []string `json:"GenerateCommands"`
	BlacklistedWords      []string `json:"BlacklistedWords"`
	AllowNonASCIIMessages bool     `json:"AllowNonAsciiMessages"`
}

// Load reads settings from the given path.
func Load(path string) (*Settings, error) {
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
