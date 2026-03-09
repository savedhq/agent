package job

import "errors"

const JobProviderSFTP Provider = "sftp"

type SFTPConfig struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password,omitempty"`
	PrivateKey string `json:"private_key,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`
	Path       string `json:"path"`
}

func (c *SFTPConfig) Validate() error {
	if c.Host == "" {
		return errors.New("host is required")
	}
	if c.Port == 0 {
		return errors.New("port is required")
	}
	if c.Username == "" {
		return errors.New("username is required")
	}
	if c.Password == "" && c.PrivateKey == "" {
		return errors.New("either password or private_key is required")
	}
	return nil
}

func (c *SFTPConfig) Type() Provider { return JobProviderSFTP }
