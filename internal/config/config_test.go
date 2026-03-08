package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfig_Defaults(t *testing.T) {
	// Create a temporary empty config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	// Create an empty file
	f, err := os.Create(configFile)
	require.NoError(t, err)
	f.Close()

	// Load config
	cfg, err := NewConfig(context.Background(), configFile)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify defaults
	assert.Equal(t, "http://localhost:8000", cfg.API)
	assert.Equal(t, "/tmp/agent", cfg.TempDir)

	// Verify Auth defaults
	assert.Equal(t, "", cfg.Auth.Server)
	assert.Equal(t, "", cfg.Auth.Username)
	assert.Equal(t, "", cfg.Auth.Password)
	assert.Equal(t, "", cfg.Auth.ClientID)
	assert.Equal(t, "", cfg.Auth.Audience)

	// Verify Path defaults
	assert.Equal(t, "git", cfg.Path.Git)
	assert.Equal(t, "mysqldump", cfg.Path.MySQL)
	assert.Equal(t, "ssh", cfg.Path.SSH)
	assert.Equal(t, "pg_dump", cfg.Path.PSQL)
	assert.Equal(t, "curl", cfg.Path.CURL)
	assert.Equal(t, "aws", cfg.Path.AWS)
	assert.Equal(t, "zip", cfg.Path.ZIP)
	assert.Equal(t, "redis-cli", cfg.Path.REDIS)
	assert.Equal(t, "tar", cfg.Path.TAR)
	assert.Equal(t, "sqlcmd", cfg.Path.MSSQL)
}
