package activities

import (
	"context"

	"go.temporal.io/sdk/activity"
)

type FileCompressionActivityInput struct {
	FilePath string `json:"file_path"`
	Provider string `json:"provider"`
}

type FileCompressionActivityOutput struct {
	FilePath string `json:"file_path"`
	Size     int64  `json:"size"`
	Checksum string `json:"checksum"`
	Name     string `json:"name"`
	MimeType string `json:"mime_type"`
}

func (a *Activities) FileCompressionActivity(ctx context.Context, input FileCompressionActivityInput) (*FileCompressionActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Debug("FileCompressionActivity called")

	result := new(FileCompressionActivityOutput)

	return result, nil
}
