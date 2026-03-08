package activities

import (
	"context"

	"go.temporal.io/sdk/activity"
)

type FileEncryptionActivityInput struct {
	FilePath string `json:"file_path"`
	Provider string `json:"provider"`
	Key      string `json:"key"`
}

type FileEncryptionActivityOutput struct {
	FilePath string `json:"file_path"`
	Size     int64  `json:"size"`
	Checksum string `json:"checksum"`
	Name     string `json:"name"`
	MimeType string `json:"mime_type"`
}

func (a *Activities) FileEncryptionActivity(ctx context.Context, input FileEncryptionActivityInput) (*FileEncryptionActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Debug("FileEncryptionActivity called")

	result := new(FileEncryptionActivityOutput)

	return result, nil
}
