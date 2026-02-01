package activities

import (
	"context"
	"os"

	"go.temporal.io/sdk/activity"
)

type CleanupActivityInput struct {
	Paths []string
}

type CleanupActivityOutput struct{}

func (a *Activities) CleanupActivity(ctx context.Context, input CleanupActivityInput) (*CleanupActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("CleanupActivity started")

	for _, path := range input.Paths {
		if path == "" {
			continue
		}
		logger.Info("Removing path", "path", path)
		if err := os.RemoveAll(path); err != nil {
			logger.Error("Failed to remove path", "path", path, "error", err)
			// Continue cleanup even if one path fails
		}
	}

	logger.Info("CleanupActivity completed")
	return &CleanupActivityOutput{}, nil
}
