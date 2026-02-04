package job

// JobProvider represents the type of backup provider
type JobProvider string

// JobConfig is the interface that all provider-specific configs must implement
type JobConfig interface {
	Validate() error
	Type() JobProvider
}

type EncryptionConfig struct {
	Enabled   bool   `json:"enabled"`
	PublicKey string `json:"public_key"`
	Algorithm string `json:"algorithm"`
}

type CompressionConfig struct {
	Enabled   bool   `json:"enabled"`
	Algorithm string `json:"algorithm"`
	Level     int    `json:"level"`
}

// Job represents a backup job configuration
type Job struct {
	ID          string            `mapstructure:"id" json:"id"`
	Provider    JobProvider       `mapstructure:"provider" json:"provider"`
	Config      JobConfig         `mapstructure:"config" json:"config"`
	Encryption  EncryptionConfig  `mapstructure:"encryption_config" json:"encryption_config"`
	Compression CompressionConfig `mapstructure:"compression_config" json:"compression_config"`
}
