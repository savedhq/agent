package activities

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.temporal.io/sdk/activity"
)

type FileUploadS3ActivityInput struct {
	FilePath  string    `json:"file_path"`
	UploadURL string    `json:"upload_url"`
	ExpiresAt time.Time `json:"expires_at"`
}

type FileUploadS3ActivityOutput struct {
	Status bool `json:"status"`
}

func (a *Activities) FileUploadS3Activity(ctx context.Context, input FileUploadS3ActivityInput) (*FileUploadS3ActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Debug("FileUploadS3Activity called", "filePath", input.FilePath)

	// Check if upload URL has expired
	if time.Now().After(input.ExpiresAt) {
		return nil, fmt.Errorf("upload URL has expired")
	}

	// Open file
	file, err := os.Open(input.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	logger.Info("Uploading file to S3", "size", fileInfo.Size())

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "PUT", input.UploadURL, file)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = fileInfo.Size()

	// Execute request
	client := &http.Client{Timeout: 60 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	result := &FileUploadS3ActivityOutput{
		Status: true,
	}

	logger.Info("File uploaded successfully to S3")
	return result, nil
}

type SyncS3BucketActivityInput struct {
	SourceRegion          string `json:"source_region"`
	SourceBucket          string `json:"source_bucket"`
	SourcePath            string `json:"source_path"`
	SourceAccessKeyID     string `json:"source_access_key_id"`
	SourceSecretAccessKey string `json:"source_secret_access_key"`
	SourceEndpoint        string `json:"source_endpoint"`
	DestRegion          string `json:"dest_region"`
	DestBucket          string `json:"dest_bucket"`
	DestPath            string `json:"dest_path"`
	DestAccessKeyID     string `json:"dest_access_key_id"`
	DestSecretAccessKey string `json:"dest_secret_access_key"`
	DestEndpoint        string `json:"dest_endpoint"`
}

type SyncS3BucketActivityOutput struct {
	Status bool `json:"status"`
}

func (a *Activities) SyncS3BucketActivity(ctx context.Context, input SyncS3BucketActivityInput) (*SyncS3BucketActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("SyncS3BucketActivity started")

	// Create a new S3 client for the source bucket
	sourceS3Client, err := newS3Client(input.SourceRegion, input.SourceEndpoint, input.SourceAccessKeyID, input.SourceSecretAccessKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create source S3 client: %w", err)
	}

	// Create a new S3 client for the destination bucket
	destS3Client, err := newS3Client(input.DestRegion, input.DestEndpoint, input.DestAccessKeyID, input.DestSecretAccessKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination S3 client: %w", err)
	}

	// List all objects in the source bucket
	listObjectsInput := &s3.ListObjectsV2Input{
		Bucket: &input.SourceBucket,
		Prefix: &input.SourcePath,
	}

	paginator := s3.NewListObjectsV2Paginator(sourceS3Client, listObjectsInput)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects in source bucket: %w", err)
		}

		for _, obj := range page.Contents {
			sourceObjectKey := *obj.Key

			// Correctly format the CopySource parameter
			copySource := fmt.Sprintf("%s/%s", input.SourceBucket, sourceObjectKey)

			// Correctly calculate the destination object key
			relativeKey, err := filepath.Rel(input.SourcePath, sourceObjectKey)
			if err != nil {
				logger.Error("failed to calculate relative path", "error", err, "sourcePath", input.SourcePath, "objectKey", sourceObjectKey)
				return nil, fmt.Errorf("failed to calculate relative path for object: %w", err)
			}
			destObjectKey := filepath.Join(input.DestPath, relativeKey)

			// Copy the object from the source to the destination bucket
			_, err = destS3Client.CopyObject(ctx, &s3.CopyObjectInput{
				Bucket:     &input.DestBucket,
				CopySource: &copySource,
				Key:        &destObjectKey,
			})
			if err != nil {
				logger.Error("failed to copy object", "error", err, "objectKey", sourceObjectKey)
				return nil, fmt.Errorf("failed to copy object: %w", err)
			}
		}
	}


	result := &SyncS3BucketActivityOutput{
		Status: true,
	}

	logger.Info("S3 bucket sync completed successfully")
	return result, nil
}

func newS3Client(region, endpoint, accessKeyID, secretAccessKey string) (*s3.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	if endpoint != "" {
		cfg.EndpointResolverWithOptions = aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
			return aws.Endpoint{URL: endpoint}, nil
		})
	}

	return s3.NewFromConfig(cfg), nil
}
