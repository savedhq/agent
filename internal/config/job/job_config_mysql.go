package job

import "fmt"

// MySQLJobConfig defines the configuration for a MySQL backup job.
type MySQLJobConfig struct {
	// ConnectionString is the database connection string.
	// Example: "user:password@tcp(122.0.0.1:3306)/dbname"
	ConnectionString string `mapstructure:"connection_string" yaml:"connection_string"`
}

func (c *MySQLJobConfig) Type() JobProvider {
	return "mysql"
}

func (c *MySQLJobConfig) Validate() error {
	if c.ConnectionString == "" {
		return fmt.Errorf("connection_string is required")
	}
	return nil
}
