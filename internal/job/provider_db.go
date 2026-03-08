package job

import "errors"

const (
	JobProviderMySQL      Provider = "mysql"
	JobProviderPostgreSQL Provider = "postgres"
	JobProviderMSSQL      Provider = "mssql"
	JobProviderRedis      Provider = "redis"
)

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

type RedisConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password,omitempty"`
	Database int    `json:"database"`
	TLS      bool   `json:"tls,omitempty"`
	Cluster  bool   `json:"cluster,omitempty"`
}

func (c *RedisConfig) Validate() error {
	if c.Host == "" {
		return errors.New("host is required")
	}
	if c.Port == 0 {
		return errors.New("port is required")
	}
	if c.Database < 0 || c.Database > 15 {
		return errors.New("database must be between 0 and 15")
	}
	return nil
}

func (c *RedisConfig) Type() Provider { return JobProviderRedis }
