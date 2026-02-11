package job

type Provider string

func (p Provider) String() string { return string(p) }

// Config is the interface that all provider-specific configs must implement
type Config interface {
	Validate() error
	Type() Provider
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

type Job struct {
	ID          string            `mapstructure:"id" json:"id"`
	Provider    Provider          `mapstructure:"provider" json:"provider"`
	Config      Config            `mapstructure:"config" json:"config"`
	Encryption  EncryptionConfig  `mapstructure:"encryption" json:"encryption"`
	Compression CompressionConfig `mapstructure:"compression" json:"compression"`
	Script      *ScriptConfig     `mapstructure:"script" json:"script"`
}
