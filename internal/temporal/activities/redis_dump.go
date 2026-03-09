package activities

import (
	"agent/internal/job"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"go.temporal.io/sdk/activity"
)

type RedisDumpActivityInput struct {
	Job *job.Job `json:"job"`
}

func (a *Activities) RedisDumpActivity(ctx context.Context, input RedisDumpActivityInput) (*DownloadActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("RedisDumpActivity started", "jobId", input.Job.ID)

	cfg, err := job.LoadAs[*job.RedisConfig](*input.Job)
	if err != nil {
		return nil, fmt.Errorf("failed to load Redis config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid Redis config: %w", err)
	}

	filename := fmt.Sprintf("%s.rdb", input.Job.ID)
	tempFilePath := filepath.Join(a.Config.TempDir, filename)

	// redis-cli -u <connection_string> --rdb <filepath>
	args := []string{"-u", cfg.ConnectionString, "--rdb", tempFilePath}

	logger.Info("Executing redis-cli --rdb", "uri", cfg.ConnectionString)

	if output, err := exec.CommandContext(ctx, "redis-cli", args...).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("redis-cli failed: %w. Output: %s", err, string(output))
	}

	file, err := os.Open(tempFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open rdb file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, fmt.Errorf("failed to calculate hash: %w", err)
	}

	fi, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	logger.Info("RedisDumpActivity completed", "filePath", tempFilePath, "size", fi.Size())

	return &DownloadActivityOutput{
		FilePath: tempFilePath,
		Size:     fi.Size(),
		Checksum: fmt.Sprintf("%x", hash.Sum(nil)),
		Name:     filename,
		MimeType: "application/x-redis-dump",
	}, nil
}
