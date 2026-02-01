package activities

import (
	"context"
	"os"

	"go.temporal.io/sdk/activity"
)

type FileCleanupActivityInput struct {
	FilePath string
}

type FileCleanupActivityOutput struct {
}

func (a *Activities) FileCleanupActivity(ctx context.Context, input FileCleanupActivityInput) (*FileCleanupActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Debug("FileCleanupActivity called")

	err := os.Remove(input.FilePath)
	if err != nil {
		logger.Error("Failed to remove file", "error", err)
		return nil, err
	}

	return &FileCleanupActivityOutput{}, nil
}
