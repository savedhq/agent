package config

import "time"

type HubConfig struct {
	Server               string        `mapstructure:"server"`
	Workspace            string        `mapstructure:"workspace"`
	Queue                string        `mapstructure:"queue"`
	ConnectionTimeout    time.Duration `mapstructure:"connection_timeout"`
	MaxReconnectAttempts int           `mapstructure:"max_reconnect_attempts"`
	TLS                  struct {
		Enabled  bool   `mapstructure:"enabled"`
		CertFile string `mapstructure:"cert_file"`
		KeyFile  string `mapstructure:"key_file"`
	} `mapstructure:"tls"`
}
