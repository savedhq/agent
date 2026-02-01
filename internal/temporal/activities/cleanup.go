package activities

import (
	"context"
	"fmt"
	"os"
)

// CleanupActivityInput defines the input for the CleanupActivity.
type CleanupActivityInput struct {
	FilePath string `json:"file_path"`
}

// CleanupActivityOutput defines the output for the CleanupActivity.
type CleanupActivityOutput struct{}

// CleanupActivity deletes a file at the given path.
func (a *Activities) CleanupActivity(ctx context.Context, input CleanupActivityInput) (*CleanupActivityOutput, error) {
	if input.FilePath == "" {
		// No file path provided, so nothing to do.
		return &CleanupActivityOutput{}, nil
	}

	err := os.Remove(input.FilePath)
	if err != nil {
		// If the file does not exist, we can consider the cleanup successful.
		if os.IsNotExist(err) {
			return &CleanupActivityOutput{}, nil
		}
		return nil, fmt.Errorf("failed to remove file %s: %w", input.FilePath, err)
	}

	return &CleanupActivityOutput{}, nil
}
