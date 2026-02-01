package activities

import (
	"agent/internal/auth"
	"agent/internal/config"
	"agent/internal/config/job"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"go.temporal.io/sdk/activity"
)

// Activities holds all activity implementations for the agent
// Refactored to use dependency injection instead of full config
type Activities struct {
	Config *config.Config
	Auth   auth.AuthService
	Hub    *config.HubConfig
}

// ExecuteScriptActivityInput defines the input for the script execution activity
type ExecuteScriptActivityInput struct {
	Job *job.Job
}

// ExecuteScriptActivityOutput defines the output for the script execution activity
type ExecuteScriptActivityOutput struct {
	OutputFile string
	Size       int64
	Checksum   string
	Name       string
}

// NewActivities creates a new Activities instance with required dependencies
func NewActivities(config *config.Config, service auth.AuthService, hubConfig config.HubConfig) *Activities {
	return &Activities{
		Config: config,
		Auth:   service,
		Hub:    &hubConfig,
	}
}

// ExecuteScriptActivity executes a local script and captures its output
func (a *Activities) ExecuteScriptActivity(ctx context.Context, input ExecuteScriptActivityInput) (*ExecuteScriptActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Executing script for job", "jobId", input.Job.ID)

	scriptConfig, ok := input.Job.Config.(*job.Script)
	if !ok {
		return nil, errors.New("invalid job config type for script provider")
	}

	// Create a temporary file to store the script's output
	outputFile, err := os.CreateTemp("", "script-output-*.log")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file for script output: %w", err)
	}
	defer outputFile.Close()

	// Create a context with the specified timeout
	execCtx, cancel := context.WithTimeout(ctx, scriptConfig.Timeout)
	defer cancel()

	// Prepare the command
	cmd := exec.CommandContext(execCtx, scriptConfig.Path)
	cmd.Stdout = outputFile
	cmd.Stderr = os.Stderr // Log stderr to the agent's stderr for now

	// Run the command
	if err := cmd.Run(); err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			logger.Error("Script execution timed out", "jobId", input.Job.ID, "timeout", scriptConfig.Timeout)
			return nil, fmt.Errorf("script execution timed out after %s", scriptConfig.Timeout)
		}
		logger.Error("Script execution failed", "jobId", input.Job.ID, "error", err)
		return nil, fmt.Errorf("script execution failed: %w", err)
	}

	// Get file info for size
	info, err := outputFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info for output file: %w", err)
	}

	// Seek to the beginning of the file to calculate the checksum
	if _, err := outputFile.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek to beginning of output file for checksum: %w", err)
	}

	hasher := sha256.New()
	if _, err := io.Copy(hasher, outputFile); err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}
	checksum := fmt.Sprintf("%x", hasher.Sum(nil))

	logger.Info("Script executed successfully", "jobId", input.Job.ID, "outputFile", outputFile.Name())

	return &ExecuteScriptActivityOutput{
		OutputFile: outputFile.Name(),
		Size:       info.Size(),
		Checksum:   checksum,
		Name:       filepath.Base(outputFile.Name()),
	}, nil
}
