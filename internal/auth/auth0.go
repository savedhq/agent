package auth

import (
	"agent/internal/config"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type auth0Service struct {
	config.AuthConfig
	mu          sync.Mutex
	cachedToken string
	tokenExpiry time.Time
}

func NewAuthService(a *config.AuthConfig) AuthService {
	return &auth0Service{
		AuthConfig: *a,
	}
}

func (as *auth0Service) Token() (string, error) {
	as.mu.Lock()
	defer as.mu.Unlock()

	// Check if we have a valid cached token (with 5 minute buffer)
	if as.cachedToken != "" && time.Now().Before(as.tokenExpiry.Add(-5*time.Minute)) {
		return as.cachedToken, nil
	}

	type tokenRequest struct {
		GrantType string `json:"grant_type"`
		Username  string `json:"username"`
		Password  string `json:"password"`
		ClientID  string `json:"client_id"`
		Audience  string `json:"audience"`
		Scope     string `json:"scope"`
		Realm     string `json:"realm"`
	}

	type tokenResponse struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}

	tokenReq := tokenRequest{
		GrantType: "http://auth0.com/oauth/grant-type/password-realm",
		Username:  as.Username,
		Password:  as.Password,
		ClientID:  as.ClientID,
		Audience:  as.Audience,
		Scope:     "profile",
		Realm:     "agent",
	}

	jsonData, err := json.Marshal(tokenReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal token request: %w", err)
	}

	url := fmt.Sprintf("%s/oauth/token", as.Server)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("auth0 returned status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp tokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal token response: %w", err)
	}

	// Cache the token and its expiry time
	as.cachedToken = tokenResp.AccessToken
	as.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return tokenResp.AccessToken, nil
}
