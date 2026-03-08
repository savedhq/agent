package activities

import (
	"agent/internal/job"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"

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
	logger.Info("DownloadActivity called", "jobId", input.Job.ID)

	httpConfig, err := job.LoadAs[*job.HTTPConfig](*input.Job)
	if err != nil {
		return nil, fmt.Errorf("failed to load HTTP config: %w", err)
	}
	if err := httpConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid HTTP config: %w", err)
	}

	parsedURL, err := url.Parse(httpConfig.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse endpoint URL: %w", err)
	}
	filename := path.Base(parsedURL.Path)
	if filename == "" || filename == "." {
		filename = "download"
	}

	if err := os.MkdirAll(a.Config.TempDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	tempFile := filepath.Join(a.Config.TempDir, fmt.Sprintf("%s-%s", input.Job.ID, filename))

	method := httpConfig.Method
	if method == "" {
		method = "GET"
	}

	args := []string{"-s", "-S", "-L", "--fail", "-o", tempFile}

	for k, vals := range httpConfig.Header {
		for _, v := range vals {
			args = append(args, "-H", fmt.Sprintf("%s: %s", k, v))
		}
	}

	if method != "GET" {
		args = append(args, "-X", method)
	}

	if httpConfig.Body != "" {
		args = append(args, "-d", httpConfig.Body)
	}

	args = append(args, httpConfig.Endpoint)

	cmd := exec.CommandContext(ctx, a.Config.Path.CURL, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("curl failed: %w, output: %s", err, string(output))
	}

	file, err := os.Open(tempFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open downloaded file: %w", err)
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat downloaded file: %w", err)
	}

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	logger.Info("DownloadActivity completed", "filePath", tempFile, "size", fi.Size())

	return &DownloadActivityOutput{
		FilePath: tempFile,
		Size:     fi.Size(),
		Checksum: fmt.Sprintf("%x", hash.Sum(nil)),
		Name:     filename,
		MimeType: "application/octet-stream",
	}, nil
}
