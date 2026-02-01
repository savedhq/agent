package job

import (
	"agent/pkg/names"
	"fmt"
)

// PostgreSQLJobConfig defines the configuration for a PostgreSQL backup job.
type PostgreSQLJobConfig struct {
	// Connection details
	Host     string `json:"host" mapstructure:"host"`
	Port     int    `json:"port" mapstructure:"port"`
	User     string `json:"user" mapstructure:"user"`
	Password string `json:"password" mapstructure:"password"`
	Database string `json:"database" mapstructure:"database"`

	// pg_dump options
	Format      string   `json:"format" mapstructure:"format"`
	SchemaOnly  bool     `json:"schema_only" mapstructure:"schema_only"`
	ExtraOptions []string `json:"extra_options" mapstructure:"extra_options"`
}

func (c *PostgreSQLJobConfig) Type() JobProvider {
	return JobProvider(names.WorkflowNamePostgreSQL)
}

func (c *PostgreSQLJobConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("host is required")
	}
	if c.User == "" {
		return fmt.Errorf("user is required")
	}
	if c.Database == "" {
		return fmt.Errorf("database is required")
	}
	return nil
}
