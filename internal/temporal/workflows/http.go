package workflows

import (
	"agent/internal/temporal/activities"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// HTTPBackupOutput defines the output for HTTP backup workflow
type HTTPBackupOutput struct {
	Status bool
}

// HTTPBackupWorkflow performs HTTP backup for agent jobs
// Agent runs on client systems and uses REST API to communicate with backend
func HTTPBackupWorkflow(ctx workflow.Context, input GeneralWorkflowInput) (*HTTPBackupOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("HTTPBackupWorkflow started", "jobId", input.JobId)
	logger.Info("HTTPBackupWorkflow started", "Provider", input.Provider)

	result := new(HTTPBackupOutput)
	// Activity options - match backend worker settings
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
	// 3. Download file from HTTP source
	////////////////////////////////////////
	DownloadActivityOutput := new(activities.DownloadActivityOutput)
	err = workflow.ExecuteActivity(
		ctx,
		"DownloadActivity",
		activities.DownloadActivityInput{
			Job: GetJobActivityOutput.Job,
		},
	).Get(ctx, DownloadActivityOutput)
	if err != nil {
		logger.Error("HTTP download failed", "error", err)
		return nil, err
	}

	logger.Info("Download completed", "filePath", DownloadActivityOutput.FilePath, "size", DownloadActivityOutput.Size)
	result.Status = true
	////////////////////////////////////////
	// 4. Compress + encrypt file
	////////////////////////////////////////
	var fileCompressionActivityOutput activities.FileCompressionActivityOutput
	err = workflow.ExecuteActivity(
		ctx,
		"FileCompressionActivity",
		activities.FileCompressionActivityInput{
			FilePath:         DownloadActivityOutput.FilePath,
			CompressionLevel: input.CompressionLevel,
		},
	).Get(ctx, &fileCompressionActivityOutput)
	if err != nil {
		logger.Error("Failed to compress file", "error", err)
		return nil, err
	}

	var getFileMetadataActivityOutput activities.GetFileMetadataActivityOutput
	err = workflow.ExecuteActivity(
		ctx,
		"GetFileMetadataActivity",
		activities.GetFileMetadataActivityInput{
			FilePath: fileCompressionActivityOutput.FilePath,
		},
	).Get(ctx, &getFileMetadataActivityOutput)
	if err != nil {
		logger.Error("Failed to get file metadata", "error", err)
		return nil, err
	}

	//var FileEncryptionActivityOutput activities.FileEncryptionActivityOutput
	//err = workflow.ExecuteActivity(
	//	ctx,
	//	"FileEncryptionActivity",
	//	activities.FileEncryptionActivityInput{},
	//).Get(ctx, &FileEncryptionActivityOutput)

	////////////////////////////////////////
	// 5. Request upload URL via API: POST /jobs/:job_id/backups/:backup_id/upload
	////////////////////////////////////////
	BackupUploadActivityOutput := new(activities.BackupUploadActivityOutput)
	err = workflow.ExecuteActivity(
		ctx,
		"BackupUploadActivity",
		activities.BackupUploadActivityInput{
			JobId:    input.JobId,
			BackupId: BackupRequestActivityOutput.ID.String(),
			FilePath: fileCompressionActivityOutput.FilePath,
			Size:     getFileMetadataActivityOutput.Size,
			Checksum: getFileMetadataActivityOutput.Checksum,
			Name:     DownloadActivityOutput.Name,
			MimeType: DownloadActivityOutput.MimeType,
		},
	).Get(ctx, BackupUploadActivityOutput)
	if err != nil {
		logger.Error("Failed to create backup upload activity", "error", err)
		return nil, err
	}

	////////////////////////////////////////
	// 6. Upload to S3 using presigned URL
	////////////////////////////////////////
	FileUploadS3ActivityOutput := new(activities.FileUploadS3ActivityOutput)
	err = workflow.ExecuteActivity(
		ctx,
		"FileUploadS3Activity",
		activities.FileUploadS3ActivityInput{
			FilePath:  DownloadActivityOutput.FilePath,
			UploadURL: BackupUploadActivityOutput.UploadURL,
			ExpiresAt: BackupUploadActivityOutput.ExpiresAt,
		},
	).Get(ctx, FileUploadS3ActivityOutput)
	if err != nil {
		logger.Error("Failed to create file upload activity", "error", err)
		return nil, err
	}

	defer func() {
		// Cleanup original file
		err := workflow.ExecuteActivity(
			ctx,
			"FileCleanupActivity",
			activities.FileCleanupActivityInput{
				FilePath: DownloadActivityOutput.FilePath,
			},
		).Get(ctx, nil)
		if err != nil {
			logger.Error("Failed to cleanup original file", "error", err)
		}

		// Only cleanup the second file if it's a different file
		if fileCompressionActivityOutput.FilePath != DownloadActivityOutput.FilePath {
			err = workflow.ExecuteActivity(
				ctx,
				"FileCleanupActivity",
				activities.FileCleanupActivityInput{
					FilePath: fileCompressionActivityOutput.FilePath,
				},
			).Get(ctx, nil)
			if err != nil {
				logger.Error("Failed to cleanup compressed file", "error", err)
			}
		}
	}()

	////////////////////////////////////////
	// 7. Confirm backup via API: POST /jobs/:job_id/backups/:backup_id/confirm
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
