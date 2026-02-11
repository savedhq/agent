package job

import (
	"errors"
	"net/http"
)

const (
	JobProviderHTTP Provider = "http"
)

type HTTPConfig struct {
	Endpoint string      `json:"endpoint"`
	Method   string      `json:"method"`
	Header   http.Header `json:"header"`
}

func (c *HTTPConfig) Validate() error {
	if c.Endpoint == "" {
		return errors.New("endpoint is required")
	}
	return nil
}

func (c *HTTPConfig) Type() Provider {
	return JobProviderHTTP
}
