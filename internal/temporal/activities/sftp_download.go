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
	"strings"

	"go.temporal.io/sdk/activity"
)

type SFTPDownloadActivityInput struct {
	Job *job.Job `json:"job"`
}

func (a *Activities) SFTPDownloadActivity(ctx context.Context, input SFTPDownloadActivityInput) (*DownloadActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("SFTPDownloadActivity started", "jobId", input.Job.ID)

	sftpConfig, err := job.LoadAs[*job.SFTPConfig](*input.Job)
	if err != nil {
		return nil, fmt.Errorf("failed to load SFTP config: %w", err)
	}
	if err := sftpConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid SFTP config: %w", err)
	}

	pathBase := filepath.Base(sftpConfig.Path)
	if pathBase == "" || pathBase == "." {
		pathBase = "download"
	}
	tempFilePath := filepath.Join(a.Config.TempDir, fmt.Sprintf("%s-%s", input.Job.ID, pathBase))

	targetURL := fmt.Sprintf("sftp://%s:%d/%s", sftpConfig.Host, sftpConfig.Port, strings.TrimPrefix(sftpConfig.Path, "/"))

	args := []string{"-s", "-S", "--fail", "-o", tempFilePath}

	if sftpConfig.PrivateKey != "" {
		keyFile, err := os.CreateTemp("", "sftp-key-*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp key file: %w", err)
		}
		defer os.Remove(keyFile.Name())
		if _, err := keyFile.WriteString(sftpConfig.PrivateKey); err != nil {
			return nil, fmt.Errorf("failed to write private key: %w", err)
		}
		keyFile.Close()
		args = append(args, "--key", keyFile.Name(), "-u", fmt.Sprintf("%s:", sftpConfig.Username))
		if sftpConfig.Passphrase != "" {
			args = append(args, "--pass", sftpConfig.Passphrase)
		}
		args = append(args, "-k")
	} else {
		args = append(args, "-u", fmt.Sprintf("%s:%s", sftpConfig.Username, sftpConfig.Password))
	}

	args = append(args, targetURL)

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

	logger.Info("SFTPDownloadActivity completed", "filePath", tempFilePath, "size", fi.Size())

	return &DownloadActivityOutput{
		FilePath: tempFilePath,
		Size:     fi.Size(),
		Checksum: fmt.Sprintf("%x", hash.Sum(nil)),
		Name:     pathBase,
		MimeType: "application/octet-stream",
	}, nil
}
