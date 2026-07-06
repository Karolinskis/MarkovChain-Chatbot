package helix

import (
	"fmt"
	"sync"
	"time"

	helixlib "github.com/nicklaw5/helix/v2"
)

const cacheTTL = 60 * time.Second

type Client struct {
	api      *helixlib.Client
	mu       sync.Mutex
	cachedAt time.Time
	cached   map[string]bool
}

func New(clientID, clientSecret string) (*Client, error) {
	api, err := helixlib.NewClient(&helixlib.Options{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	})
	if err != nil {
		return nil, fmt.Errorf("create helix client: %w", err)
	}

	resp, err := api.RequestAppAccessToken([]string{})
	if err != nil {
		return nil, fmt.Errorf("get app access token: %w", err)
	}
	api.SetAppAccessToken(resp.Data.AccessToken)

	return &Client{api: api}, nil
}

// LiveChannels returns a map of channel name to live status.
// Results are cached for 60 seconds; repeated calls within that window
// return the cached value without hitting the API.
func (c *Client) LiveChannels(channels []string) (map[string]bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cached != nil && time.Since(c.cachedAt) < cacheTTL {
		return c.cached, nil
	}

	result := make(map[string]bool, len(channels))
	for _, ch := range channels {
		result[ch] = false
	}

	resp, err := c.api.GetStreams(&helixlib.StreamsParams{
		UserLogins: channels,
	})
	if err != nil {
		return nil, fmt.Errorf("get streams: %w", err)
	}
	if resp.ErrorMessage != "" {
		return nil, fmt.Errorf("helix: %s", resp.ErrorMessage)
	}

	for _, stream := range resp.Data.Streams {
		result[stream.UserLogin] = true
	}

	c.cached = result
	c.cachedAt = time.Now()
	return result, nil
}
