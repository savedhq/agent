package config

import (
	"agent/internal/config/job"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config represents unified configuration for both backend server and worker
type Config struct {
	API     string     `mapstructure:"api"` // Backend API URL for fetching hub config
	TempDir string     `mapstructure:"temp_dir"`
	Auth    AuthConfig `mapstructure:"auth"`
	Jobs    []job.Job  `mapstructure:"jobs"`
	Log     LogConfig  `mapstructure:"log"`
}

// tempConfig is used for initial Viper unmarshaling with map configs
type tempConfig struct {
	API     string     `mapstructure:"api"`
	TempDir string     `mapstructure:"temp_dir"`
	Auth    AuthConfig `mapstructure:"auth"`
	Jobs    []tempJob  `mapstructure:"jobs"`
	Log     LogConfig  `mapstructure:"log"`
}

// tempJob is used for initial unmarshaling before converting to typed config
type tempJob struct {
	ID          string                `mapstructure:"id"`
	Provider    string                `mapstructure:"provider"`
	Config      map[string]any        `mapstructure:"config"`
	Encryption  job.EncryptionConfig  `mapstructure:"encryption_config"`
	Compression job.CompressionConfig `mapstructure:"compression_config"`
}

// NewConfig loads configuration from file, environment variables, and optionally Vault
// configPath: path to the config file (e.g., "config.yaml"). If empty, looks for "config.yaml" in current directory
func NewConfig(ctx context.Context, configPath string) (*Config, error) {
	// Set default values matching config.yaml
	viper.SetDefault("api", "")

	// Log configuration defaults
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.path", "stdout")
	viper.SetDefault("log.max_size", 5)
	viper.SetDefault("log.max_backups", 3)
	viper.SetDefault("log.max_age", 28)
	viper.SetDefault("log.compress", true)

	// Auth configuration defaults
	viper.SetDefault("auth.server", "")
	viper.SetDefault("auth.username", "")
	viper.SetDefault("auth.password", "")
	viper.SetDefault("auth.client_id", "")
	viper.SetDefault("auth.audience", "")

	// Set config file
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
	}

	// Read the config file
	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			fmt.Println("No config file found, using defaults and environment variables")
		} else {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Set up Viper to read from environment variables
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// First unmarshal into temp config with map-based job configs
	var temp tempConfig
	if err := viper.Unmarshal(&temp); err != nil {
		return nil, err
	}

	// Convert to final config with typed job configs
	config := &Config{
		API:     temp.API,
		TempDir: temp.TempDir,
		Auth:    temp.Auth,
		Jobs:    make([]job.Job, len(temp.Jobs)),
		Log:     temp.Log,
	}

	// Convert each job's map config to typed config
	for i, tj := range temp.Jobs {
		provider := job.JobProvider(tj.Provider)

		// Convert map to typed config
		typedConfig, err := job.FromMap(provider, tj.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to load config for job %s (provider %s): %w", tj.ID, tj.Provider, err)
		}

		config.Jobs[i] = job.Job{
			ID:          tj.ID,
			Provider:    provider,
			Config:      typedConfig,
			Encryption:  tj.Encryption,
			Compression: tj.Compression,
		}
	}

	return config, nil
}
