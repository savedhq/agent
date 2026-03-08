package activities

import (
	"context"
	"os"

	"go.temporal.io/sdk/activity"
)

type FileCleanupActivityInput struct {
	FilePath string `json:"file_path"`
}

func (a *Activities) FileCleanupActivity(ctx context.Context, input FileCleanupActivityInput) error {
	logger := activity.GetLogger(ctx)
	logger.Info("FileCleanupActivity called", "filePath", input.FilePath)

	err := os.Remove(input.FilePath)
	if err != nil {
		logger.Error("Failed to remove file", "error", err)
		return err
	}

	return nil
}
