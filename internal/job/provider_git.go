package job

import "errors"

const (
	JobProviderGit Provider = "git"
)

type GitConfig struct {
	URL        string `json:"url"`
	Branch     string `json:"branch,omitempty"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
	PrivateKey string `json:"private_key,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`
	Depth      int    `json:"depth,omitempty"`
	Submodules bool   `json:"submodules,omitempty"`
}

func (c *GitConfig) Validate() error {
	if c.URL == "" {
		return errors.New("url is required")
	}
	return nil
}

func (c *GitConfig) Type() Provider { return JobProviderGit }
