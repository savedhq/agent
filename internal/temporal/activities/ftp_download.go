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

	protocol := strings.ToLower(ftpConfig.Protocol)
	if protocol == "" {
		protocol = "ftp"
	}

	remotePath := ftpConfig.Path
	pathBase := filepath.Base(remotePath)
	isDirectory := strings.HasSuffix(remotePath, "/") || pathBase == "" || pathBase == "." || pathBase == "/"

	lftpURL := fmt.Sprintf("%s://%s:%d", protocol, ftpConfig.Host, ftpConfig.Port)

	var tempFilePath, name, mimeType string

	if isDirectory {
		dirName := filepath.Base(strings.TrimRight(remotePath, "/"))
		if dirName == "" || dirName == "." || dirName == "/" {
			dirName = "backup"
		}

		mirrorDir, err := os.MkdirTemp(a.Config.TempDir, "ftp-mirror-*")
		if err != nil {
			return nil, fmt.Errorf("failed to create mirror temp dir: %w", err)
		}
		defer os.RemoveAll(mirrorDir)

		mirrorRemote := remotePath
		if mirrorRemote == "" {
			mirrorRemote = "/"
		}

		lftpScript := fmt.Sprintf(
			"open -u %s,%s %s; mirror %s %s; exit",
			ftpConfig.Username, ftpConfig.Password, lftpURL, mirrorRemote, mirrorDir,
		)
		if out, err := exec.CommandContext(ctx, "lftp", "-c", lftpScript).CombinedOutput(); err != nil {
			return nil, fmt.Errorf("lftp mirror failed: %w, output: %s", err, string(out))
		}

		archiveName := fmt.Sprintf("%s-%s.tar.gz", input.Job.ID, dirName)
		tempFilePath = filepath.Join(a.Config.TempDir, archiveName)
		if out, err := exec.CommandContext(ctx, "tar", "-czf", tempFilePath, "-C", mirrorDir, ".").CombinedOutput(); err != nil {
			return nil, fmt.Errorf("failed to archive mirrored directory: %w, output: %s", err, string(out))
		}

		name, mimeType = archiveName, "application/gzip"
	} else {
		tempFilePath = filepath.Join(a.Config.TempDir, fmt.Sprintf("%s-%s", input.Job.ID, pathBase))

		lftpScript := fmt.Sprintf(
			"open -u %s,%s %s; get %s -o %s; exit",
			ftpConfig.Username, ftpConfig.Password, lftpURL, remotePath, tempFilePath,
		)
		if out, err := exec.CommandContext(ctx, "lftp", "-c", lftpScript).CombinedOutput(); err != nil {
			return nil, fmt.Errorf("transfer failed: %w, output: %s", err, string(out))
		}

		name, mimeType = pathBase, "application/octet-stream"
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
		Name:     name,
		MimeType: mimeType,
	}, nil
}
