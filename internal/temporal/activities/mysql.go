package activities

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/go-sql-driver/mysql"
	"go.temporal.io/sdk/activity"
)

type MySQLDumpInput struct {
	ConnectionString string
	OutputPath       string
}

type MySQLDumpOutput struct {
	Path string
}

func (a *Activities) MySQLDumpActivity(ctx context.Context, input MySQLDumpInput) (*MySQLDumpOutput, error) {
	log := activity.GetLogger(ctx)
	log.Info("running mysqldump")

	// Ensure the output directory exists
	if err := os.MkdirAll(a.Config.TempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Parse the DSN
	config, err := mysql.ParseDSN(input.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse DSN: %w", err)
	}

	// Build the mysqldump command
	args := []string{
		"--user=" + config.User,
		"--password=" + config.Passwd,
		"--host=" + strings.Split(config.Addr, ":")[0],
		"--port=" + strings.Split(config.Addr, ":")[1],
		config.DBName,
	}
	// #nosec
	cmd := exec.Command("mysqldump", args...)
	outputFile, err := os.Create(input.OutputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()
	cmd.Stdout = outputFile

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("mysqldump failed: %w", err)
	}

	log.Info("mysqldump finished successfully", "path", input.OutputPath)

	return &MySQLDumpOutput{Path: input.OutputPath}, nil
}
