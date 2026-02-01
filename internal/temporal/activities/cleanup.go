package activities

import (
	"context"
	"fmt"
	"os"

	"go.temporal.io/sdk/activity"
)

type CleanupActivityInput struct {
	Path string
}

func (a *Activities) CleanupActivity(ctx context.Context, input CleanupActivityInput) error {
	logger := activity.GetLogger(ctx)
	logger.Info("CleanupActivity started", "path", input.Path)

	if input.Path == "" {
		logger.Warn("CleanupActivity received an empty path, nothing to do.")
		return nil
	}

	err := os.RemoveAll(input.Path)
	if err != nil {
		logger.Error("Failed to clean up file", "path", input.Path, "error", err)
		return fmt.Errorf("failed to remove path %s: %w", input.Path, err)
	}

	logger.Info("CleanupActivity completed successfully", "path", input.Path)
	return nil
}
