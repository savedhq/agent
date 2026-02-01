package activities

import (
	"agent/internal/config/job"
	"context"
	"fmt"
	"os"
	"os/exec"
)

// PostgreSQLDumpActivity defines the activity for running pg_dump.
func (a *Activities) PostgreSQLDumpActivity(ctx context.Context, jobConfig job.PostgreSQLJobConfig) (string, error) {
	// Create a temporary file to store the dump
	dumpFile, err := os.CreateTemp("", "postgresql-dump-*.sql")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary dump file: %w", err)
	}
	defer dumpFile.Close()

	// Construct the pg_dump command
	args := []string{
		"-h", jobConfig.Host,
		"-p", fmt.Sprintf("%d", jobConfig.Port),
		"-U", jobConfig.User,
		"-d", jobConfig.Database,
		"-f", dumpFile.Name(),
	}

	if jobConfig.Format != "" {
		args = append(args, "--format", jobConfig.Format)
	}

	if jobConfig.SchemaOnly {
		args = append(args, "--schema-only")
	}

	args = append(args, jobConfig.ExtraOptions...)

	cmd := exec.Command("pg_dump", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", jobConfig.Password))

	// Execute the command
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("pg_dump failed: %s: %w", string(output), err)
	}

	return dumpFile.Name(), nil
}
