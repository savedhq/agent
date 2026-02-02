package activities

import (
	"context"
	"fmt"
	"os"

	"go.temporal.io/sdk/activity"
)

type CreateTempDirInput struct {
	Pattern string
}

type CreateTempDirOutput struct {
	Path string
}

func (a *Activities) CreateTempDirActivity(ctx context.Context, input CreateTempDirInput) (*CreateTempDirOutput, error) {
	log := activity.GetLogger(ctx)
	log.Info("creating temp directory")

	path, err := os.MkdirTemp("", input.Pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	return &CreateTempDirOutput{Path: path}, nil
}

type RemoveFileInput struct {
	Path string
}

type RemoveFileOutput struct {
}

func (a *Activities) RemoveFileActivity(ctx context.Context, input RemoveFileInput) (*RemoveFileOutput, error) {
	log := activity.GetLogger(ctx)
	log.Info("removing file", "path", input.Path)

	if err := os.Remove(input.Path); err != nil {
		return nil, fmt.Errorf("failed to remove file: %w", err)
	}

	return &RemoveFileOutput{}, nil
}
