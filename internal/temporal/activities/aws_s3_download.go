package activities

import (
	"agent/internal/job"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.temporal.io/sdk/activity"
)

type AWSS3DownloadActivityInput struct {
	Job *job.Job `json:"job"`
}

func (a *Activities) AWSS3DownloadActivity(ctx context.Context, input AWSS3DownloadActivityInput) (*DownloadActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("AWSS3DownloadActivity started", "jobId", input.Job.ID)

	s3Config, err := job.LoadAs[*job.AWSS3Config](*input.Job)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS S3 config: %w", err)
	}
	if err := s3Config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid AWS S3 config: %w", err)
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(s3Config.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(s3Config.AccessKeyID, s3Config.SecretAccessKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load aws config: %w", err)
	}
	if s3Config.Endpoint != "" {
		cfg.BaseEndpoint = aws.String(s3Config.Endpoint)
	}

	client := s3.NewFromConfig(cfg)

	objects, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(s3Config.Bucket),
		Prefix: aws.String(s3Config.Path),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}
	if len(objects.Contents) == 0 {
		return nil, fmt.Errorf("no objects found at path: %s", s3Config.Path)
	}

	isDir := strings.HasSuffix(s3Config.Path, "/")

	// Single file exact match
	if len(objects.Contents) == 1 && *objects.Contents[0].Key == s3Config.Path && !isDir {
		return a.s3DownloadSingleFile(ctx, client, s3Config, input.Job.ID, *objects.Contents[0].Key)
	}

	return a.s3DownloadAsZip(ctx, client, s3Config, input.Job.ID, objects.Contents)
}

func (a *Activities) s3DownloadSingleFile(ctx context.Context, client *s3.Client, cfg *job.AWSS3Config, jobID, key string) (*DownloadActivityOutput, error) {
	fileName := filepath.Base(key)
	tempFilePath := filepath.Join(a.Config.TempDir, fmt.Sprintf("%s-%s", jobID, fileName))

	file, err := os.Create(tempFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer file.Close()

	downloader := manager.NewDownloader(client)
	if _, err = downloader.Download(ctx, file, &s3.GetObjectInput{
		Bucket: aws.String(cfg.Bucket),
		Key:    aws.String(key),
	}); err != nil {
		return nil, fmt.Errorf("failed to download object: %w", err)
	}

	return a.hashAndReturn(tempFilePath, fileName, "application/octet-stream")
}
