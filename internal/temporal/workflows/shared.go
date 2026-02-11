package workflows

import (
	"agent/internal"
	"agent/internal/job"
	"agent/internal/temporal/activities"

	"go.temporal.io/sdk/workflow"
)

// ProcessAndUpload handles compress → encrypt → upload → confirm steps shared by all providers.
func ProcessAndUpload(ctx workflow.Context, j *job.Job, jobId, backupId, filePath string, size int64, checksum, name, mimeType string) error {
	logger := workflow.GetLogger(ctx)

	// Compress
	currentFile := filePath
	if j.Compression.Enabled {
		var out activities.FileCompressionActivityOutput
		err := workflow.ExecuteActivity(ctx, internal.ActivityNameCompressFile,
			activities.FileCompressionActivityInput{FilePath: currentFile, Provider: j.Compression.Algorithm},
		).Get(ctx, &out)
		if err != nil {
			return err
		}
		workflow.ExecuteActivity(ctx, internal.ActivityNameFileCleanup,
			activities.FileCleanupActivityInput{FilePath: currentFile}).Get(ctx, nil)
		currentFile = out.FilePath
	}

	// Encrypt
	if j.Encryption.Enabled {
		var out activities.FileEncryptionActivityOutput
		err := workflow.ExecuteActivity(ctx, internal.ActivityNameEncryptFile,
			activities.FileEncryptionActivityInput{FilePath: currentFile, Provider: j.Encryption.Algorithm, Key: j.Encryption.PublicKey},
		).Get(ctx, &out)
		if err != nil {
			return err
		}
		workflow.ExecuteActivity(ctx, internal.ActivityNameFileCleanup,
			activities.FileCleanupActivityInput{FilePath: currentFile}).Get(ctx, nil)
		currentFile = out.FilePath
	}

	// Request upload URL
	var uploadOut activities.BackupUploadActivityOutput
	err := workflow.ExecuteActivity(ctx, internal.ActivityNameBackupUpload,
		activities.BackupUploadActivityInput{
			JobId: jobId, BackupId: backupId, FilePath: filePath,
			Size: size, Checksum: checksum, Name: name, MimeType: mimeType,
		},
	).Get(ctx, &uploadOut)
	if err != nil {
		return err
	}

	// Upload to S3
	err = workflow.ExecuteActivity(ctx, internal.ActivityNameFileUploadS3,
		activities.FileUploadS3ActivityInput{
			FilePath: currentFile, UploadURL: uploadOut.UploadURL, ExpiresAt: uploadOut.ExpiresAt,
		},
	).Get(ctx, nil)
	workflow.ExecuteActivity(ctx, internal.ActivityNameFileCleanup,
		activities.FileCleanupActivityInput{FilePath: currentFile}).Get(ctx, nil)
	if err != nil {
		return err
	}

	// Confirm backup
	err = workflow.ExecuteActivity(ctx, internal.ActivityNameBackupConfirm,
		activities.BackupConfirmActivityInput{JobId: jobId, BackupId: backupId, Status: true},
	).Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to confirm backup", "error", err)
		return err
	}

	return nil
}
