package activities

import (
	"agent/internal/job"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"go.temporal.io/sdk/activity"
)

type PostgreSQLDumpActivityInput struct {
	Job *job.Job `json:"job"`
}

func (a *Activities) PostgreSQLDumpActivity(ctx context.Context, input PostgreSQLDumpActivityInput) (*DownloadActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("PostgreSQLDumpActivity started", "jobId", input.Job.ID)

	cfg, err := job.LoadAs[*job.PostgreSQLConfig](*input.Job)
	if err != nil {
		return nil, fmt.Errorf("failed to load PostgreSQL config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid PostgreSQL config: %w", err)
	}

	filename := fmt.Sprintf("%s.sql", input.Job.ID)
	tempFilePath := filepath.Join(a.Config.TempDir, filename)

	file, err := os.Create(tempFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	args := []string{cfg.ConnectionString, "--no-owner", "--no-acl"}

	cmd := exec.CommandContext(ctx, "pg_dump", args...)
	hash := sha256.New()
	cmd.Stdout = io.MultiWriter(file, hash)

	if err := cmd.Run(); err != nil {
		file.Close()
		return nil, fmt.Errorf("pg_dump failed: %w", err)
	}
	file.Close()

	fi, err := os.Stat(tempFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat temp file: %w", err)
	}

	logger.Info("PostgreSQLDumpActivity completed", "filePath", tempFilePath, "size", fi.Size())

	return &DownloadActivityOutput{
		FilePath: tempFilePath,
		Size:     fi.Size(),
		Checksum: fmt.Sprintf("%x", hash.Sum(nil)),
		Name:     filename,
		MimeType: "application/sql",
	}, nil
}
