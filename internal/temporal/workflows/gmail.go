package workflows

import (
	"agent/internal/temporal/activities"
	"agent/pkg/names"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// GmailBackupOutput defines the output for Gmail backup workflow
type GmailBackupOutput struct {
	Status bool
}

// GmailBackupWorkflow performs Gmail backup for agent jobs
func GmailBackupWorkflow(ctx workflow.Context, input GeneralWorkflowInput) (*GmailBackupOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("GmailBackupWorkflow started", "jobId", input.JobId)

	result := &GmailBackupOutput{Status: false}
	// Activity options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 1 * time.Hour,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    5 * time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	////////////////////////////////////////
	// 1. Get job details
	////////////////////////////////////////
	var getJobOutput activities.GetJobActivityOutput
	err := workflow.ExecuteActivity(
		ctx,
		names.ActivityNameGetJob,
		activities.GetJobActivityInput{JobId: input.JobId},
	).Get(ctx, &getJobOutput)
	if err != nil {
		logger.Error("Failed to get job details", "error", err)
		return nil, err
	}

	logger.Info("Job details fetched", "jobId", input.JobId, "provider", input.Provider)

	////////////////////////////////////////
	// 2. Create backup entity via API
	////////////////////////////////////////
	var backupRequestOutput activities.BackupRequestActivityOutput
	err = workflow.ExecuteActivity(
		ctx,
		names.ActivityNameBackupRequest,
		activities.BackupRequestActivityInput{Job: getJobOutput.Job},
	).Get(ctx, &backupRequestOutput)
	if err != nil {
		logger.Error("Failed to create backup request", "error", err)
		return nil, err
	}

	logger.Info("Backup created successfully", "backupID", backupRequestOutput.ID.String())

	////////////////////////////////////////
	// 3. Export Gmail emails
	////////////////////////////////////////
	var exportOutput activities.GmailExportActivityOutput
	err = workflow.ExecuteActivity(
		ctx,
		names.ActivityNameGmailExport,
		activities.GmailExportActivityInput{
			Job: getJobOutput.Job,
		},
	).Get(ctx, &exportOutput)
	if err != nil {
		logger.Error("Gmail export failed", "error", err)
		return nil, err
	}

	logger.Info("Export completed", "filePath", exportOutput.FilePath, "size", exportOutput.Size)
	result.Status = true
	currentFilePath := exportOutput.FilePath
	currentSize := exportOutput.Size
	currentChecksum := exportOutput.Checksum

	////////////////////////////////////////
	// 4. Compress + encrypt file
	////////////////////////////////////////
	fileModified := false
	if getJobOutput.Job.Compression.Enabled {
		var compressionOutput activities.FileCompressionActivityOutput
		err = workflow.ExecuteActivity(ctx, names.ActivityNameCompressFile,
			activities.FileCompressionActivityInput{
				FilePath: currentFilePath,
				Provider: getJobOutput.Job.Compression.Algorithm,
			}).Get(ctx, &compressionOutput)
		if err != nil {
			return nil, err
		}
		// Cleanup original file if it was different
		if compressionOutput.FilePath != currentFilePath {
			workflow.ExecuteActivity(ctx, names.ActivityNameFileCleanup, activities.FileCleanupActivityInput{FilePath: currentFilePath}).Get(ctx, nil)
			fileModified = true
		}
		currentFilePath = compressionOutput.FilePath
	}

	if getJobOutput.Job.Encryption.Enabled {
		var encryptionOutput activities.FileEncryptionActivityOutput
		err = workflow.ExecuteActivity(ctx, names.ActivityNameEncryptFile,
			activities.FileEncryptionActivityInput{
				FilePath: currentFilePath,
				Provider: getJobOutput.Job.Encryption.Algorithm,
				Key:      getJobOutput.Job.Encryption.PublicKey,
			}).Get(ctx, &encryptionOutput)
		if err != nil {
			return nil, err
		}
		// Cleanup previous file if it was different
		if encryptionOutput.FilePath != currentFilePath {
			workflow.ExecuteActivity(ctx, names.ActivityNameFileCleanup, activities.FileCleanupActivityInput{FilePath: currentFilePath}).Get(ctx, nil)
			fileModified = true
		}
		currentFilePath = encryptionOutput.FilePath
	}

	// If file was modified by compression or encryption, we need to recalculate metadata
	if fileModified {
		var metadataOutput activities.FileMetadataActivityOutput
		err = workflow.ExecuteActivity(ctx, names.ActivityNameFileMetadata, activities.FileMetadataActivityInput{
			FilePath: currentFilePath,
		}).Get(ctx, &metadataOutput)
		if err != nil {
			logger.Error("Failed to recalculate file metadata", "error", err)
			return nil, err
		}
		currentSize = metadataOutput.Size
		currentChecksum = metadataOutput.Checksum
	}

	////////////////////////////////////////
	// 5. Request upload URL
	////////////////////////////////////////
	var backupUploadOutput activities.BackupUploadActivityOutput
	err = workflow.ExecuteActivity(
		ctx,
		names.ActivityNameBackupUpload,
		activities.BackupUploadActivityInput{
			JobId:    input.JobId,
			BackupId: backupRequestOutput.ID.String(),
			FilePath: currentFilePath,
			Size:     currentSize,
			Checksum: currentChecksum,
			Name:     exportOutput.Name,
			MimeType: exportOutput.MimeType,
		},
	).Get(ctx, &backupUploadOutput)
	if err != nil {
		logger.Error("Failed to create backup upload request", "error", err)
		return nil, err
	}

	////////////////////////////////////////
	// 6. Upload to S3
	////////////////////////////////////////
	err = workflow.ExecuteActivity(
		ctx,
		names.ActivityNameFileUploadS3,
		activities.FileUploadS3ActivityInput{
			FilePath:  currentFilePath,
			UploadURL: backupUploadOutput.UploadURL,
			ExpiresAt: backupUploadOutput.ExpiresAt,
		},
	).Get(ctx, nil)

	// Final cleanup
	workflow.ExecuteActivity(ctx, names.ActivityNameFileCleanup, activities.FileCleanupActivityInput{FilePath: currentFilePath}).Get(ctx, nil)

	if err != nil {
		logger.Error("S3 upload failed", "error", err)
		return nil, err
	}

	////////////////////////////////////////
	// 7. Confirm backup
	////////////////////////////////////////
	err = workflow.ExecuteActivity(
		ctx,
		names.ActivityNameBackupConfirm,
		activities.BackupConfirmActivityInput{
			JobId:    input.JobId,
			BackupId: backupRequestOutput.ID.String(),
			Status:   result.Status,
		},
	).Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to confirm backup", "error", err)
		return nil, err
	}

	return result, nil
}
