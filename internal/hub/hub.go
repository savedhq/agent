package hub

import (
	"agent/internal/authentication"
	"agent/internal/config"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LoadHubConfig fetches hub configuration from the backend API
// Must be called after loading config file and obtaining auth token
func LoadHubConfig(ctx context.Context, as authentication.AuthenticationService, c *config.Config) (*config.HubConfig, error) {

	url := fmt.Sprintf("%s/v1/agents/config/hub", c.API)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	token, err := as.Token()
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var hubConfig *config.HubConfig
	if err := json.Unmarshal(body, &hubConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal hub config response: %w", err)
	}

	return hubConfig, nil
}
