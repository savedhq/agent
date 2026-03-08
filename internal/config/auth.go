package config

type AuthConfig struct {
	Server   string `mapstructure:"server"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	ClientID string `mapstructure:"client_id"`
	Audience string `mapstructure:"audience"`
}
