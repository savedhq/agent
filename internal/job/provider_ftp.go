package job

import "errors"

const (
	JobProviderFTP Provider = "ftp"
)

type FTPConfig struct {
	Protocol   string `json:"protocol"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	Path       string `json:"path"`
	PrivateKey string `json:"private_key,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`
}

func (c *FTPConfig) Validate() error {
	if c.Host == "" {
		return errors.New("host is required")
	}
	if c.Port == 0 {
		return errors.New("port is required")
	}
	if c.Username == "" {
		return errors.New("username is required")
	}
	if c.Protocol == "" {
		c.Protocol = "ftp"
	}
	if c.Protocol == "sftp" {
		if c.Password == "" && c.PrivateKey == "" {
			return errors.New("either password or private_key is required for sftp")
		}
	} else if c.Password == "" {
		return errors.New("password is required")
	}
	return nil
}

func (c *FTPConfig) Type() Provider { return JobProviderFTP }
