package config

import (
	"agent/internal/job"
	"context"
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	API     string     `mapstructure:"api"`
	TempDir string     `mapstructure:"temp_dir"`
	Auth    AuthConfig `mapstructure:"auth"`
	Path    PathConfig `mapstructure:"path"`
	Jobs    []job.Job  `mapstructure:"jobs"`
}

func NewConfig(_ context.Context, configPath string) (*Config, error) {
	if configPath == "" {
		configPath = "config.yaml"
	}

	v := viper.New()
	v.SetConfigFile(configPath)

	// API defaults
	v.SetDefault("api", "http://localhost:8000")

	// Global defaults
	v.SetDefault("temp_dir", "/tmp/agent")

	// Auth defaults
	v.SetDefault("auth.server", "")
	v.SetDefault("auth.username", "")
	v.SetDefault("auth.password", "")
	v.SetDefault("auth.client_id", "")
	v.SetDefault("auth.audience", "")

	// Path defaults
	v.SetDefault("path.git", "git")
	v.SetDefault("path.mysql", "mysqldump")
	v.SetDefault("path.ssh", "ssh")
	v.SetDefault("path.psql", "pg_dump")
	v.SetDefault("path.curl", "curl")
	v.SetDefault("path.aws", "aws")
	v.SetDefault("path.zip", "zip")
	v.SetDefault("path.redis", "redis-cli")
	v.SetDefault("path.tar", "tar")
	v.SetDefault("path.mssql", "sqlcmd")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	// rawJob mirrors Job but keeps Config as a raw map for two-pass parsing
	type rawJob struct {
		ID          string                `mapstructure:"id"`
		Provider    job.Provider          `mapstructure:"provider"`
		Config      map[string]any        `mapstructure:"config"`
		Encryption  job.EncryptionConfig  `mapstructure:"encryption"`
		Compression job.CompressionConfig `mapstructure:"compression"`
	}

	// First pass: unmarshal with raw config maps
	var raw struct {
		API     string     `mapstructure:"api"`
		TempDir string     `mapstructure:"temp_dir"`
		Auth    AuthConfig `mapstructure:"auth"`
		Path    PathConfig `mapstructure:"path"`
		Jobs    []rawJob   `mapstructure:"jobs"`
	}
	if err := v.Unmarshal(&raw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Second pass: convert raw config maps to typed configs
	cfg := &Config{
		API:     raw.API,
		TempDir: raw.TempDir,
		Auth:    raw.Auth,
		Path:    raw.Path,
		Jobs:    make([]job.Job, 0, len(raw.Jobs)),
	}

	for _, rj := range raw.Jobs {
		typedCfg, err := job.ConfigFromMap(rj.Provider, rj.Config)
		if err != nil {
			return nil, fmt.Errorf("job %s: %w", rj.ID, err)
		}
		cfg.Jobs = append(cfg.Jobs, job.Job{
			ID:          rj.ID,
			Provider:    rj.Provider,
			Config:      typedCfg,
			Encryption:  rj.Encryption,
			Compression: rj.Compression,
		})
	}

	return cfg, nil
}
