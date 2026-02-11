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
	"path/filepath"
	"strconv"

	"go.temporal.io/sdk/activity"
)

type GitDownloadActivityInput struct {
	Job *job.Job `json:"job"`
}

func (a *Activities) GitDownloadActivity(ctx context.Context, input GitDownloadActivityInput) (*DownloadActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("GitDownloadActivity started", "jobId", input.Job.ID)

	cfg, err := job.LoadAs[*job.GitConfig](*input.Job)
	if err != nil {
		return nil, fmt.Errorf("failed to load Git config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid Git config: %w", err)
	}

	cloneDir, err := os.MkdirTemp("", "git-clone-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp clone dir: %w", err)
	}
	defer os.RemoveAll(cloneDir)

	args := []string{"clone"}
	if cfg.Depth > 0 {
		args = append(args, "--depth", strconv.Itoa(cfg.Depth))
	}
	if cfg.Branch != "" {
		args = append(args, "--branch", cfg.Branch)
	}
	if cfg.Submodules {
		args = append(args, "--recurse-submodules")
	}

	targetURL := cfg.URL
	var env []string

	if cfg.PrivateKey != "" {
		keyFile, err := os.CreateTemp("", "git-key-*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp key file: %w", err)
		}
		defer os.Remove(keyFile.Name())
		if _, err := keyFile.WriteString(cfg.PrivateKey); err != nil {
			return nil, fmt.Errorf("failed to write private key: %w", err)
		}
		keyFile.Close()
		if err := os.Chmod(keyFile.Name(), 0600); err != nil {
			return nil, fmt.Errorf("failed to chmod key file: %w", err)
		}
		sshCmd := fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no", keyFile.Name())
		env = append(os.Environ(), fmt.Sprintf("GIT_SSH_COMMAND=%s", sshCmd))
	} else if cfg.Username != "" && cfg.Password != "" {
		u, err := url.Parse(cfg.URL)
		if err != nil {
			return nil, fmt.Errorf("invalid git URL: %w", err)
		}
		u.User = url.UserPassword(cfg.Username, cfg.Password)
		targetURL = u.String()
	}

	args = append(args, targetURL, cloneDir)
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = env

	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("git clone failed: %w, output: %s", err, string(output))
	}

	archiveName := fmt.Sprintf("%s.tar.gz", input.Job.ID)
	tempFilePath := filepath.Join(a.Config.TempDir, archiveName)

	tarCmd := exec.CommandContext(ctx, "tar", "-czf", tempFilePath, "-C", cloneDir, ".")
	if output, err := tarCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("failed to create archive: %w, output: %s", err, string(output))
	}

	file, err := os.Open(tempFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open archive: %w", err)
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

	logger.Info("GitDownloadActivity completed", "filePath", tempFilePath, "size", fi.Size())

	return &DownloadActivityOutput{
		FilePath: tempFilePath,
		Size:     fi.Size(),
		Checksum: fmt.Sprintf("%x", hash.Sum(nil)),
		Name:     archiveName,
		MimeType: "application/gzip",
	}, nil
}
