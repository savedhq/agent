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

type ScriptRunActivityInput struct {
	Job *job.Job
}

type ScriptRunActivityOutput struct {
	FilePath string
	Size     int64
	Checksum string
	Name     string
	MimeType string
}

func (a *Activities) ScriptRunActivity(ctx context.Context, input ScriptRunActivityInput) (*ScriptRunActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Starting ScriptRunActivity", "jobID", input.Job.ID)

	if input.Job.Script == nil {
		return nil, fmt.Errorf("script config is missing")
	}

	config := input.Job.Script

	// Create a temp file to store the stdout
	tmpFile, err := os.CreateTemp("", "backup-script-*.dat")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	cmd := exec.CommandContext(ctx, config.Command, config.Args...)
	if config.WorkDir != "" {
		cmd.Dir = config.WorkDir
	}

	// Capture stdout to file
	cmd.Stdout = tmpFile

	// Capture stderr for logging
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	// Read stderr
	stderrOutput, _ := io.ReadAll(stderrPipe)

	if err := cmd.Wait(); err != nil {
		logger.Error("Command execution failed", "error", err, "stderr", string(stderrOutput))
		return nil, fmt.Errorf("command execution failed: %s: %w", string(stderrOutput), err)
	}

	// Calculate size and checksum
	info, err := tmpFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat temp file: %w", err)
	}

	if _, err := tmpFile.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek temp file: %w", err)
	}

	hash := sha256.New()
	if _, err := io.Copy(hash, tmpFile); err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	logger.Info("ScriptRunActivity completed successfully", "filePath", tmpFile.Name(), "size", info.Size())

	return &ScriptRunActivityOutput{
		FilePath: tmpFile.Name(),
		Size:     info.Size(),
		Checksum: fmt.Sprintf("%x", hash.Sum(nil)),
		Name:     filepath.Base(tmpFile.Name()),
		MimeType: "application/octet-stream",
	}, nil
}
