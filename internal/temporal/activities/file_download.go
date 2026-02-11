package activities

import (
	"agent/internal/job"
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
	Job *job.Job `json:"job"`
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

	httpConfig, err := job.LoadAs[*job.HTTPConfig](*input.Job)
	if err != nil {
		return nil, fmt.Errorf("failed to load HTTP config: %w", err)
	}
	if err := httpConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid HTTP config: %w", err)
	}

	endpoint := httpConfig.Endpoint
	method := httpConfig.Method
	if method == "" {
		method = "GET"
	}

	logger.Info("Downloading file", "endpoint", endpoint, "method", method)

	req, err := http.NewRequestWithContext(ctx, method, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for k, vals := range httpConfig.Header {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}

	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	filename := filepath.Base(endpoint)
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		if _, params, err := mime.ParseMediaType(cd); err == nil {
			if fn := params["filename"]; fn != "" {
				filename = fn
			}
		}
	}

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
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

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

	logger.Info("Download completed", "filePath", result.FilePath, "size", result.Size)
	return result, nil
}
