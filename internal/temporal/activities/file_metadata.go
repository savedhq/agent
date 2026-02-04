package activities

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"

	"go.temporal.io/sdk/activity"
)

type FileMetadataActivityInput struct {
	FilePath string `json:"file_path"`
}

type FileMetadataActivityOutput struct {
	Size     int64  `json:"size"`
	Checksum string `json:"checksum"`
}

func (a *Activities) FileMetadataActivity(ctx context.Context, input FileMetadataActivityInput) (*FileMetadataActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Debug("FileMetadataActivity called", "filePath", input.FilePath)

	file, err := os.Open(input.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	h := sha256.New()
	if _, err := io.Copy(h, file); err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	result := &FileMetadataActivityOutput{
		Size:     fi.Size(),
		Checksum: fmt.Sprintf("%x", h.Sum(nil)),
	}

	logger.Info("File metadata calculated", "size", result.Size, "checksum", result.Checksum)
	return result, nil
}
