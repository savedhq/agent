package activities

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"

	"go.temporal.io/sdk/activity"
)

type GetFileMetadataActivityInput struct {
	FilePath string
}

type GetFileMetadataActivityOutput struct {
	Size     int64
	Checksum string
}

func (a *Activities) GetFileMetadataActivity(ctx context.Context, input GetFileMetadataActivityInput) (*GetFileMetadataActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Debug("GetFileMetadataActivity called")

	file, err := os.Open(input.FilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return nil, err
	}

	return &GetFileMetadataActivityOutput{
		Size:     stat.Size(),
		Checksum: hex.EncodeToString(hasher.Sum(nil)),
	}, nil
}
