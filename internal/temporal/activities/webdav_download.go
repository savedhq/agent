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

type WebDAVDownloadActivityInput struct {
	Job *job.Job `json:"job"`
}

func (a *Activities) WebDAVDownloadActivity(ctx context.Context, input WebDAVDownloadActivityInput) (*DownloadActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("WebDAVDownloadActivity started", "jobId", input.Job.ID)

	cfg, err := job.LoadAs[*job.WebDAVConfig](*input.Job)
	if err != nil {
		return nil, fmt.Errorf("failed to load WebDAV config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid WebDAV config: %w", err)
	}

	u, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid WebDAV URL: %w", err)
	}
	if cfg.Path != "" {
		u.Path = path.Join(u.Path, cfg.Path)
	}

	fileName := filepath.Base(u.Path)
	if fileName == "" || fileName == "." || fileName == "/" {
		fileName = "download"
	}
	tempFilePath := filepath.Join(a.Config.TempDir, fmt.Sprintf("%s-%s", input.Job.ID, fileName))

	args := []string{
		"-u", fmt.Sprintf("%s:%s", cfg.Username, cfg.Password),
		"-s", "-S", "--fail", "-L",
		"-o", tempFilePath,
		u.String(),
	}

	output, err := exec.CommandContext(ctx, "curl", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("transfer failed: %w, output: %s", err, string(output))
	}

	file, err := os.Open(tempFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open downloaded file: %w", err)
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, fmt.Errorf("failed to calculate hash: %w", err)
	}

	logger.Info("WebDAVDownloadActivity completed", "filePath", tempFilePath, "size", fi.Size())

	return &DownloadActivityOutput{
		FilePath: tempFilePath,
		Size:     fi.Size(),
		Checksum: fmt.Sprintf("%x", hash.Sum(nil)),
		Name:     fileName,
		MimeType: "application/octet-stream",
	}, nil
}
