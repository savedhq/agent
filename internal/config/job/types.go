package job

// JobProvider represents the type of backup provider
type JobProvider string

// JobConfig is the interface that all provider-specific configs must implement
type JobConfig interface {
	Validate() error
	Type() JobProvider
}

// Job represents a backup job configuration
type Job struct {
	ID       string      `mapstructure:"id"`
	Provider JobProvider `mapstructure:"provider"`
	Config   JobConfig   `mapstructure:"config"`
}
