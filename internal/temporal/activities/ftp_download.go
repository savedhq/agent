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

type FTPDownloadActivityInput struct {
	Job *job.Job `json:"job"`
}

func (a *Activities) FTPDownloadActivity(ctx context.Context, input FTPDownloadActivityInput) (*DownloadActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("FTPDownloadActivity started", "jobId", input.Job.ID)

	ftpConfig, err := job.LoadAs[*job.FTPConfig](*input.Job)
	if err != nil {
		return nil, fmt.Errorf("failed to load FTP config: %w", err)
	}
	if err := ftpConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid FTP config: %w", err)
	}

	pathBase := filepath.Base(ftpConfig.Path)
	if pathBase == "" || pathBase == "." {
		pathBase = "download"
	}
	tempFilePath := filepath.Join(a.Config.TempDir, fmt.Sprintf("%s-%s", input.Job.ID, pathBase))

	protocol := strings.ToLower(ftpConfig.Protocol)
	if protocol == "" {
		protocol = "ftp"
	}
	targetURL := fmt.Sprintf("%s://%s:%d/%s", protocol, ftpConfig.Host, ftpConfig.Port, strings.TrimPrefix(ftpConfig.Path, "/"))

	args := []string{"-s", "-S", "--fail", "--ftp-create-dirs", "-o", tempFilePath}

	if protocol == "sftp" && ftpConfig.PrivateKey != "" {
		keyFile, err := os.CreateTemp("", "sftp-key-*")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp key file: %w", err)
		}
		defer os.Remove(keyFile.Name())
		if _, err := keyFile.WriteString(ftpConfig.PrivateKey); err != nil {
			return nil, fmt.Errorf("failed to write private key: %w", err)
		}
		keyFile.Close()
		args = append(args, "--key", keyFile.Name(), "-u", fmt.Sprintf("%s:", ftpConfig.Username))
		if ftpConfig.Passphrase != "" {
			args = append(args, "--pass", ftpConfig.Passphrase)
		}
		args = append(args, "-k")
	} else {
		args = append(args, "-u", fmt.Sprintf("%s:%s", ftpConfig.Username, ftpConfig.Password))
		if protocol == "ftps" {
			args = append(args, "--ssl-reqd", "-k")
		}
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

	logger.Info("FTPDownloadActivity completed", "filePath", tempFilePath, "size", fi.Size())

	return &DownloadActivityOutput{
		FilePath: tempFilePath,
		Size:     fi.Size(),
		Checksum: fmt.Sprintf("%x", hash.Sum(nil)),
		Name:     pathBase,
		MimeType: "application/octet-stream",
	}, nil
}
