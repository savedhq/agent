package job

import "errors"

const JobProviderPostgreSQL Provider = "postgres"

type PostgreSQLConfig struct {
	ConnectionString string `json:"connection_string"`
	Format           string `json:"format,omitempty"`
	SchemaOnly       bool   `json:"schema_only,omitempty"`
	DataOnly         bool   `json:"data_only,omitempty"`
}

func (c *PostgreSQLConfig) Validate() error {
	if c.ConnectionString == "" {
		return errors.New("connection_string is required")
	}
	return nil
}

func (c *PostgreSQLConfig) Type() Provider { return JobProviderPostgreSQL }
