package workflows

import (
	"agent/internal/temporal/activities"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type AWSS3BackupOutput struct {
	Status bool
}

func AWSS3BackupWorkflow(ctx workflow.Context, input GeneralWorkflowInput) (*AWSS3BackupOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("AWSS3BackupWorkflow started", "jobId", input.JobId)
	logger.Info("AWSS3BackupWorkflow started", "Provider", input.Provider)

	result := new(AWSS3BackupOutput)
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    5 * time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	////////////////////////////////////////
	// 1. Get job details from config.yaml
	////////////////////////////////////////
	GetJobActivityOutput := new(activities.GetJobActivityOutput)

	activity := workflow.ExecuteActivity(
		ctx,
		"GetJobActivity",
		activities.GetJobActivityInput{JobId: input.JobId},
	)
	if err := activity.Get(ctx, GetJobActivityOutput); err != nil {
		logger.Error("Failed to get job details", "error", err)
		return nil, err
	}

	logger.Info("Job details fetched", "jobId", input.JobId, "provider", input.Provider)

	////////////////////////////////////////
	// 2. Create backup entity via API: POST /jobs/:job_id/request
	////////////////////////////////////////
	BackupRequestActivityOutput := new(activities.BackupRequestActivityOutput)

	err := workflow.ExecuteActivity(
		ctx,
		"BackupRequestActivity",
		activities.BackupRequestActivityInput{Job: GetJobActivityOutput.Job},
	).Get(ctx, BackupRequestActivityOutput)
	if err != nil {
		logger.Error("Failed to create backup request", "error", err)
		return nil, err
	}

	logger.Info("Backup created successfully", "backupID", BackupRequestActivityOutput.ID.String())

	////////////////////////////////////////
	// 3. Download file from S3 source
	////////////////////////////////////////
	S3DownloadActivityOutput := new(activities.S3DownloadActivityOutput)
	err = workflow.ExecuteActivity(
		ctx,
		"S3DownloadActivity",
		activities.S3DownloadActivityInput{
			Job: GetJobActivityOutput.Job,
		},
	).Get(ctx, S3DownloadActivityOutput)
	if err != nil {
		logger.Error("S3 download failed", "error", err)
		return nil, err
	}

	logger.Info("Download completed", "filePath", S3DownloadActivityOutput.FilePath, "size", S3DownloadActivityOutput.Size)
	result.Status = true

	// Track file path through compression/encryption
	filePath := S3DownloadActivityOutput.FilePath

	////////////////////////////////////////
	// 4. Compress file (if enabled)
	////////////////////////////////////////
	if GetJobActivityOutput.Job.Compression.Enabled {
		FileCompressionActivityOutput := new(activities.FileCompressionActivityOutput)
		err = workflow.ExecuteActivity(
			ctx,
			"FileCompressionActivity",
			activities.FileCompressionActivityInput{
				InputPath:  filePath,
				OutputPath: filePath + ".gz",
			},
		).Get(ctx, FileCompressionActivityOutput)
		if err != nil {
			logger.Error("Failed to compress file", "error", err)
			return nil, err
		}
		filePath = FileCompressionActivityOutput.OutputPath
		logger.Info("Compression completed", "filePath", filePath)
	}

	////////////////////////////////////////
	// 5. Encrypt file (if enabled)
	////////////////////////////////////////
	if GetJobActivityOutput.Job.Encryption.Enabled {
		FileEncryptionActivityOutput := new(activities.FileEncryptionActivityOutput)
		err = workflow.ExecuteActivity(
			ctx,
			"FileEncryptionActivity",
			activities.FileEncryptionActivityInput{
				InputPath:  filePath,
				OutputPath: filePath + ".enc",
			},
		).Get(ctx, FileEncryptionActivityOutput)
		if err != nil {
			logger.Error("Failed to encrypt file", "error", err)
			return nil, err
		}
		filePath = FileEncryptionActivityOutput.OutputPath
		logger.Info("Encryption completed", "filePath", filePath)
	}

	////////////////////////////////////////
	// 6. Request upload URL via API: POST /jobs/:job_id/backups/:backup_id/upload
	////////////////////////////////////////
	BackupUploadActivityOutput := new(activities.BackupUploadActivityOutput)
	err = workflow.ExecuteActivity(
		ctx,
		"BackupUploadActivity",
		activities.BackupUploadActivityInput{
			JobId:    input.JobId,
			BackupId: BackupRequestActivityOutput.ID.String(),
			FilePath: filePath,
			Size:     S3DownloadActivityOutput.Size,
			Checksum: S3DownloadActivityOutput.Checksum,
			Name:     S3DownloadActivityOutput.Name,
			MimeType: S3DownloadActivityOutput.MimeType,
		},
	).Get(ctx, BackupUploadActivityOutput)
	if err != nil {
		logger.Error("Failed to create backup upload activity", "error", err)
		return nil, err
	}

	////////////////////////////////////////
	// 7. Upload to S3 using presigned URL
	////////////////////////////////////////
	FileUploadS3ActivityOutput := new(activities.FileUploadS3ActivityOutput)
	err = workflow.ExecuteActivity(
		ctx,
		"FileUploadS3Activity",
		activities.FileUploadS3ActivityInput{
			FilePath:  filePath,
			UploadURL: BackupUploadActivityOutput.UploadURL,
			ExpiresAt: BackupUploadActivityOutput.ExpiresAt,
		},
	).Get(ctx, FileUploadS3ActivityOutput)
	if err != nil {
		logger.Error("Failed to create file upload activity", "error", err)
		return nil, err
	}

	////////////////////////////////////////
	// 8. Confirm backup via API: POST /jobs/:job_id/backups/:backup_id/confirm
	////////////////////////////////////////
	BackupConfirmActivityOutput := new(activities.BackupConfirmActivityOutput)
	err = workflow.ExecuteActivity(
		ctx,
		"BackupConfirmActivity",
		activities.BackupConfirmActivityInput{
			JobId:    input.JobId,
			BackupId: BackupRequestActivityOutput.ID.String(),
			Status:   result.Status,
		},
	).Get(ctx, BackupConfirmActivityOutput)
	if err != nil {
		logger.Error("Failed to create backup confirm activity", "error", err)
		return nil, err
	}

	return result, nil
}
