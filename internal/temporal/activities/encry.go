package activities

import (
	"context"

	"go.temporal.io/sdk/activity"
)

type FileEncryptionActivityInput struct {
	InputPath  string
	OutputPath string
}

type FileEncryptionActivityOutput struct {
	OutputPath string
}

func (a *Activities) FileEncryptionActivity(ctx context.Context, input FileEncryptionActivityInput) (*FileEncryptionActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Debug("FileEncryptionActivity called")

	result := new(FileEncryptionActivityOutput)

	return result, nil
}
