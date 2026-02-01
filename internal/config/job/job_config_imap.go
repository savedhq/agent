package job

import (
	"errors"
)

const (
	JobProviderIMAP JobProvider = "imap"
)

type IMAPConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	TLS      bool   `json:"tls"`
}

func (c *IMAPConfig) Validate() error {
	if c.Host == "" {
		return errors.New("host is required")
	}
	if c.Port == 0 {
		return errors.New("port is required")
	}
	if c.User == "" {
		return errors.New("user is required")
	}
	if c.Password == "" {
		return errors.New("password is required")
	}
	return nil
}

func (c *IMAPConfig) Type() JobProvider {
	return JobProviderIMAP
}
