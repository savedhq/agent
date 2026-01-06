package config

// Job represents a backup job configuration
type Job struct {
	ID       string                 `mapstructure:"id"`
	Provider string                 `mapstructure:"provider"`
	Config   map[string]interface{} `mapstructure:"config"`
}
