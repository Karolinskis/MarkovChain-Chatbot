package helix

import (
	"fmt"
	"strings"

	helixlib "github.com/nicklaw5/helix/v2"
)

// Helix allows at most 100 user logins per Get Streams request.
const maxLoginsPerRequest = 100

type Client struct {
	api *helixlib.Client
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

// LiveChannels returns a map of lowercase channel login to live status.
// Every requested channel is present in the result; channels not currently
// streaming map to false.
func (c *Client) LiveChannels(channels []string) (map[string]bool, error) {
	result := make(map[string]bool, len(channels))
	logins := make([]string, 0, len(channels))
	for _, ch := range channels {
		login := strings.ToLower(ch)
		if _, ok := result[login]; !ok {
			result[login] = false
			logins = append(logins, login)
		}
	}

	for start := 0; start < len(logins); start += maxLoginsPerRequest {
		batch := logins[start:min(start+maxLoginsPerRequest, len(logins))]
		resp, err := c.api.GetStreams(&helixlib.StreamsParams{
			UserLogins: batch,
			First:      len(batch),
		})
		if err != nil {
			return nil, fmt.Errorf("get streams: %w", err)
		}
		if resp.ErrorMessage != "" {
			return nil, fmt.Errorf("helix: %s", resp.ErrorMessage)
		}

		for _, stream := range resp.Data.Streams {
			result[strings.ToLower(stream.UserLogin)] = true
		}
	}

	return result, nil
}
