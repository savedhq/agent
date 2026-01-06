package activities

import (
	"agent/internal/config"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"go.temporal.io/sdk/activity"
)

type DownloadActivityInput struct {
	Job *config.Job `json:"job"`
}

type DownloadActivityOutput struct {
	Size     int64  `json:"size"`
	Checksum string `json:"checksum"`
	Name     string `json:"name"`
	MimeType string `json:"mime_type"`
	FilePath string `json:"file_path"`
}

func (a *Activities) DownloadActivity(ctx context.Context, input DownloadActivityInput) (*DownloadActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Debug("DownloadActivity called", "jobId", input.Job.ID)

	// Get Endpoint from job config
	endpoint, ok := input.Job.Config["endpoint"].(string)
	if !ok || endpoint == "" {
		return nil, fmt.Errorf("endpoint not found in job config")
	}

	// Get Method from job config
	method, ok := input.Job.Config["method"].(string)
	if !ok || method == "" {
		method = "GET"
	}

	logger.Info("Downloading file", "endpoint", endpoint, "method", method)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, method, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers if present
	if headers, ok := input.Job.Config["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			if strVal, ok := v.(string); ok {
				req.Header.Set(k, strVal)
			}
		}
	}

	// Get timeout
	timeoutVal := 30 * time.Minute
	if t, ok := input.Job.Config["timeout"].(float64); ok {
		timeoutVal = time.Duration(t) * time.Second
	} else if t, ok := input.Job.Config["timeout"].(int); ok {
		timeoutVal = time.Duration(t) * time.Second
	}

	// Execute request
	client := &http.Client{Timeout: timeoutVal}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Extract filename from URL or Content-Disposition header
	filename := filepath.Base(endpoint)
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if _, params, err := mime.ParseMediaType(cd); err == nil {
			if fn := params["filename"]; fn != "" {
				filename = fn
			}
		}
	}

	// Create temp file
	tempFile := filepath.Join(a.Config.TempDir, fmt.Sprintf("%s-%s", input.Job.ID, filename))
	file, err := os.Create(tempFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer file.Close()

	// Create hash calculator
	hash := sha256.New()
	multiWriter := io.MultiWriter(file, hash)

	// Download and calculate hash
	size, err := io.Copy(multiWriter, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	// Get MIME type
	mimeType := resp.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	result := &DownloadActivityOutput{
		FilePath: tempFile,
		Size:     size,
		Checksum: fmt.Sprintf("%x", hash.Sum(nil)),
		Name:     filename,
		MimeType: mimeType,
	}

	logger.Info("Download completed", "filePath", result.FilePath, "size", result.Size, "checksum", result.Checksum)
	return result, nil
}
