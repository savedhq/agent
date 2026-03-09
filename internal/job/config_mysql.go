package job

import "errors"

const JobProviderMySQL Provider = "mysql"

type MySQLConfig struct {
	ConnectionString string `json:"connection_string"`
}

func (c *MySQLConfig) Validate() error {
	if c.ConnectionString == "" {
		return errors.New("connection_string is required")
	}
	return nil
}

func (c *MySQLConfig) Type() Provider { return JobProviderMySQL }
