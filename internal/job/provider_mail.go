package job

import "errors"

const (
	JobProviderIMAP Provider = "imap"
)

type IMAPConfig struct {
	Host     string   `json:"host"`
	Port     int      `json:"port"`
	Username string   `json:"username"`
	Password string   `json:"password"`
	TLS      bool     `json:"tls,omitempty"`
	Mailbox  string   `json:"mailbox,omitempty"`
	Folders  []string `json:"folders,omitempty"`
	Since    string   `json:"since,omitempty"`
	Timeout  int      `json:"timeout,omitempty"`
}

func (c *IMAPConfig) Validate() error {
	if c.Host == "" {
		return errors.New("host is required")
	}
	if c.Port == 0 {
		return errors.New("port is required")
	}
	if c.Username == "" {
		return errors.New("username is required")
	}
	if c.Password == "" {
		return errors.New("password is required")
	}
	return nil
}

func (c *IMAPConfig) Type() Provider { return JobProviderIMAP }
