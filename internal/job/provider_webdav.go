package job

import "errors"

const (
	JobProviderWebDAV Provider = "webdav"
)

type WebDAVConfig struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
	Path     string `json:"path,omitempty"`
	TLS      bool   `json:"tls,omitempty"`
	Timeout  int    `json:"timeout,omitempty"`
}

func (c *WebDAVConfig) Validate() error {
	if c.URL == "" {
		return errors.New("url is required")
	}
	if c.Username == "" {
		return errors.New("username is required")
	}
	if c.Password == "" {
		return errors.New("password is required")
	}
	return nil
}

func (c *WebDAVConfig) Type() Provider { return JobProviderWebDAV }
