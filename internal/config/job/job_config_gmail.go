package job

import (
	"agent/pkg/names"
	"errors"
)

const (
	JobProviderGmail JobProvider = JobProvider(names.WorkflowNameGmail)
)

type GmailConfig struct {
	Email        string `json:"email"`         // Google account email
	ClientID     string `json:"client_id"`     // OAuth2 client ID
	ClientSecret string `json:"client_secret"` // OAuth2 client secret
	RefreshToken string `json:"refresh_token"` // OAuth2 refresh token
	Format       string `json:"format"`       // "mbox" or "eml"
	Query        string `json:"query"`        // Gmail search query
}

func (c *GmailConfig) Validate() error {
	if c.Email == "" {
		return errors.New("email is required")
	}
	if c.ClientID == "" {
		return errors.New("client_id is required")
	}
	if c.ClientSecret == "" {
		return errors.New("client_secret is required")
	}
	if c.RefreshToken == "" {
		return errors.New("refresh_token is required")
	}
	if c.Format != "mbox" && c.Format != "eml" {
		return errors.New("format must be either 'mbox' or 'eml'")
	}
	return nil
}

func (c *GmailConfig) Type() JobProvider {
	return JobProviderGmail
}
