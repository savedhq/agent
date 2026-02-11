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

type MySQLDumpActivityInput struct {
	Job *job.Job `json:"job"`
}

func (a *Activities) MySQLDumpActivity(ctx context.Context, input MySQLDumpActivityInput) (*DownloadActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("MySQLDumpActivity started", "jobId", input.Job.ID)

	cfg, err := job.LoadAs[*job.MySQLConfig](*input.Job)
	if err != nil {
		return nil, fmt.Errorf("failed to load MySQL config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid MySQL config: %w", err)
	}

	// Parse connection string: user:password@tcp(host:port)/dbname
	connStr := cfg.ConnectionString
	atIndex := strings.LastIndex(connStr, "@")
	if atIndex == -1 {
		return nil, fmt.Errorf("invalid connection string format: missing @")
	}
	userPass := connStr[:atIndex]
	authParts := strings.SplitN(userPass, ":", 2)
	username := authParts[0]
	password := ""
	if len(authParts) > 1 {
		password = authParts[1]
	}

	remaining := connStr[atIndex+1:]
	hostStart := strings.Index(remaining, "(")
	hostEnd := strings.Index(remaining, ")")
	if hostStart == -1 || hostEnd == -1 {
		return nil, fmt.Errorf("invalid connection string format: missing protocol/host")
	}
	hostPort := remaining[hostStart+1 : hostEnd]
	hostParts := strings.Split(hostPort, ":")
	host := hostParts[0]
	port := "3306"
	if len(hostParts) > 1 {
		port = hostParts[1]
	}
	slashIndex := strings.Index(remaining[hostEnd:], "/")
	if slashIndex == -1 {
		return nil, fmt.Errorf("invalid connection string format: missing database name")
	}
	dbName := remaining[hostEnd+slashIndex+1:]
	if qIndex := strings.Index(dbName, "?"); qIndex != -1 {
		dbName = dbName[:qIndex]
	}

	filename := fmt.Sprintf("%s.sql", input.Job.ID)
	tempFilePath := filepath.Join(a.Config.TempDir, filename)

	file, err := os.Create(tempFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	args := []string{"-h", host, "-P", port, "-u", username}
	if password != "" {
		args = append(args, fmt.Sprintf("-p%s", password))
	}
	args = append(args, "--single-transaction", "--quick", "--lock-tables=false", "--routines", "--triggers", dbName)

	cmd := exec.CommandContext(ctx, "mysqldump", args...)
	hash := sha256.New()
	cmd.Stdout = io.MultiWriter(file, hash)

	var stderr strings.Builder
	cmd.Stderr = &stderr

	logger.Info("Executing mysqldump", "host", host, "port", port, "db", dbName)

	if err := cmd.Run(); err != nil {
		file.Close()
		return nil, fmt.Errorf("mysqldump failed: %w. Stderr: %s", err, stderr.String())
	}
	file.Close()

	fi, err := os.Stat(tempFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat temp file: %w", err)
	}

	logger.Info("MySQLDumpActivity completed", "filePath", tempFilePath, "size", fi.Size())

	return &DownloadActivityOutput{
		FilePath: tempFilePath,
		Size:     fi.Size(),
		Checksum: fmt.Sprintf("%x", hash.Sum(nil)),
		Name:     filename,
		MimeType: "application/sql",
	}, nil
}
