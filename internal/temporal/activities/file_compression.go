package activities

import (
	"context"

	"go.temporal.io/sdk/activity"
)

type FileCompressionActivityInput struct {
	InputPath  string
	OutputPath string
}

type FileCompressionActivityOutput struct {
	OutputPath string
}

func (a *Activities) FileCompressionActivity(ctx context.Context, input FileCompressionActivityInput) (*FileCompressionActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Debug("FileCompressionActivity called")

	result := new(FileCompressionActivityOutput)

	return result, nil
}
