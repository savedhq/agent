package config

import (
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
	Jobs    []Job      `mapstructure:"jobs"`
}

// NewConfig loads configuration from file, environment variables, and optionally Vault
// configPath: path to the config file (e.g., "config.yaml"). If empty, looks for "config.yaml" in current directory
func NewConfig(ctx context.Context, configPath string) (*Config, error) {
	config := new(Config)

	// Set default values matching config.yaml
	viper.SetDefault("api", "")

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

	if err := viper.Unmarshal(config); err != nil {
		return nil, err
	}

	return config, nil
}
