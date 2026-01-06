package activities

import (
	"context"

	"go.temporal.io/sdk/activity"
)

type FileCompressionActivityInput struct {
}

type FileCompressionActivityOutput struct {
}

func (a *Activities) FileCompressionActivity(ctx context.Context, input FileCompressionActivityInput) (*FileCompressionActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Debug("FileCompressionActivity called")

	result := new(FileCompressionActivityOutput)

	return result, nil
}
