package config

type HubConfig struct {
	Server    string `mapstructure:"server"`
	Workspace string `mapstructure:"workspace"`
	Queue     string `mapstructure:"queue"`
	TLS       struct {
		Enabled bool `mapstructure:"enabled"`
	} `mapstructure:"tls"`
}
