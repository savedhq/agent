package job

import "errors"

const JobProviderMSSQL Provider = "mssql"

type MSSQLConfig struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Database    string `json:"database"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	Instance    string `json:"instance,omitempty"`
	Encrypt     bool   `json:"encrypt,omitempty"`
	TrustCert   bool   `json:"trust_cert,omitempty"`
	ConnTimeout int    `json:"conn_timeout,omitempty"`
}

func (c *MSSQLConfig) Validate() error {
	if c.Host == "" {
		return errors.New("host is required")
	}
	if c.Port == 0 {
		return errors.New("port is required")
	}
	if c.Database == "" {
		return errors.New("database is required")
	}
	if c.Username == "" {
		return errors.New("username is required")
	}
	if c.Password == "" {
		return errors.New("password is required")
	}
	return nil
}

func (c *MSSQLConfig) Type() Provider { return JobProviderMSSQL }
