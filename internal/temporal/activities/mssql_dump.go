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

type MSSQLDumpActivityInput struct {
	Job *job.Job `json:"job"`
}

func (a *Activities) MSSQLDumpActivity(ctx context.Context, input MSSQLDumpActivityInput) (*DownloadActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("MSSQLDumpActivity started", "jobId", input.Job.ID)

	cfg, err := job.LoadAs[*job.MSSQLConfig](*input.Job)
	if err != nil {
		return nil, fmt.Errorf("failed to load MSSQL config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid MSSQL config: %w", err)
	}

	tempDir, err := os.MkdirTemp("", "mssql-backup-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	backupFileName := fmt.Sprintf("backup-%s.bak", input.Job.ID)
	backupFilePath := filepath.Join(tempDir, backupFileName)

	query := fmt.Sprintf("BACKUP DATABASE [%s] TO DISK = N'%s' WITH NOFORMAT, NOINIT, NAME = N'full-backup', SKIP, NOREWIND, NOUNLOAD, STATS = 10",
		cfg.Database, backupFilePath)

	args := []string{
		"-S", fmt.Sprintf("%s,%d", cfg.Host, cfg.Port),
		"-d", cfg.Database,
		"-Q", query,
	}

	if cfg.Username != "" {
		args = append(args, "-U", cfg.Username)
		if cfg.Password != "" {
			args = append(args, "-P", cfg.Password)
		}
	} else {
		args = append(args, "-E")
	}
	if cfg.Encrypt {
		args = append(args, "-N")
	}
	if cfg.TrustCert {
		args = append(args, "-C")
	}

	output, err := exec.CommandContext(ctx, "sqlcmd", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("mssql backup failed: %w, output: %s", err, string(output))
	}

	archiveName := fmt.Sprintf("%s.tar.gz", input.Job.ID)
	archivePath := filepath.Join(a.Config.TempDir, archiveName)

	if err := exec.CommandContext(ctx, "tar", "-czf", archivePath, "-C", tempDir, backupFileName).Run(); err != nil {
		return nil, fmt.Errorf("failed to create archive: %w", err)
	}

	file, err := os.Open(archivePath)
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

	logger.Info("MSSQLDumpActivity completed", "filePath", archivePath, "size", fi.Size())

	return &DownloadActivityOutput{
		FilePath: archivePath,
		Size:     fi.Size(),
		Checksum: fmt.Sprintf("%x", hash.Sum(nil)),
		Name:     archiveName,
		MimeType: "application/gzip",
	}, nil
}
