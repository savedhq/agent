package activities

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

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
