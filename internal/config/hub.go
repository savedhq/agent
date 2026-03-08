package config

type HubConfig struct {
	Server    string       `mapstructure:"server"`
	Workspace string       `mapstructure:"workspace"`
	Queue     string       `mapstructure:"queue"`
	TLS       HubTLSConfig `mapstructure:"tls"`
}

type HubTLSConfig struct {
	Enabled bool `mapstructure:"enabled"`
}
