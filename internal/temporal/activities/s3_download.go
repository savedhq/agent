package activities

import (
	"agent/internal/config/job"
	agents3 "agent/pkg/s3"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.temporal.io/sdk/activity"
)

type S3DownloadActivityInput struct {
	Job *job.Job `json:"job"`
}

type S3DownloadActivityOutput struct {
	Size     int64  `json:"size"`
	Checksum string `json:"checksum"`
	Name     string `json:"name"`
	MimeType string `json:"mime_type"`
	FilePath string `json:"file_path"`
}

func (a *Activities) S3DownloadActivity(ctx context.Context, input S3DownloadActivityInput) (*S3DownloadActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Debug("S3DownloadActivity called", "jobId", input.Job.ID)

	s3Config, err := job.LoadAs[*job.AWSS3Config](*input.Job)
	if err != nil {
		return nil, fmt.Errorf("failed to load S3 config: %w", err)
	}

	if err := s3Config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid S3 config: %w", err)
	}

	client, err := agents3.NewClient(s3Config.Region, s3Config.Endpoint, s3Config.AccessKeyID, s3Config.SecretAccessKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	logger.Info("Downloading from S3", "bucket", s3Config.Bucket, "path", s3Config.Path)

	resp, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &s3Config.Bucket,
		Key:    &s3Config.Path,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get S3 object: %w", err)
	}
	defer resp.Body.Close()

	filename := filepath.Base(s3Config.Path)
	tempFile := filepath.Join(a.Config.TempDir, fmt.Sprintf("%s-%s", input.Job.ID, filename))
	file, err := os.Create(tempFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	multiWriter := io.MultiWriter(file, hash)

	size, err := io.Copy(multiWriter, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to download S3 object: %w", err)
	}

	mimeType := "application/octet-stream"
	if resp.ContentType != nil {
		mimeType = *resp.ContentType
	}

	result := &S3DownloadActivityOutput{
		FilePath: tempFile,
		Size:     size,
		Checksum: fmt.Sprintf("%x", hash.Sum(nil)),
		Name:     filename,
		MimeType: mimeType,
	}

	logger.Info("S3 download completed", "filePath", result.FilePath, "size", result.Size)
	return result, nil
}
