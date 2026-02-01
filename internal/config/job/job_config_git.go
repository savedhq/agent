package job

import (
	"errors"
)

const (
	JobProviderGit JobProvider = "git"
)

type GitAuthConfig struct {
	// SSHPrivateKey is the private key for SSH authentication
	SSHPrivateKey string `json:"ssh_private_key,omitempty"`
	// HTTPSUsername is the username for HTTPS authentication
	HTTPSUsername string `json:"https_username,omitempty"`
	// HTTPSPassword is the password or token for HTTPS authentication
	HTTPSPassword string `json:"https_password,omitempty"`
}

type GitConfig struct {
	URL        string         `json:"url"`
	Shallow    bool           `json:"shallow,omitempty"`
	Submodules bool           `json:"submodules,omitempty"`
	Auth       *GitAuthConfig `json:"auth,omitempty"`
}

func (c *GitConfig) Validate() error {
	if c.URL == "" {
		return errors.New("url is required")
	}
	return nil
}

func (c *GitConfig) Type() JobProvider {
	return JobProviderGit
}
