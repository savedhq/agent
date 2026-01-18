package job

import (
	"errors"
)

const (
	JobProviderHTTP JobProvider = "http"
)

type HTTPConfig struct {
	URL     string            `json:"url"`
	Method  string            `json:"method,omitempty"`  // GET, POST, etc.
	Headers map[string]string `json:"headers,omitempty"` // Custom headers
}

func (c *HTTPConfig) Validate() error {
	if c.URL == "" {
		return errors.New("url is required")
	}
	return nil
}

func (c *HTTPConfig) Type() JobProvider {
	return JobProviderHTTP
}
