package workflows

import (
	"agent/internal/temporal/activities"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// IMAPBackupOutput defines the output for IMAP backup workflow
type IMAPBackupOutput struct {
	Status bool
}

// IMAPBackupWorkflow performs IMAP backup for agent jobs
func IMAPBackupWorkflow(ctx workflow.Context, input GeneralWorkflowInput) (*IMAPBackupOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("IMAPBackupWorkflow started", "jobId", input.JobId)
	logger.Info("IMAPBackupWorkflow started", "Provider", input.Provider)

	result := new(IMAPBackupOutput)
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

	// Defer cleanup activity to ensure it runs regardless of workflow success or failure
	var cleanupPath string
	defer func() {
		if cleanupPath != "" {
			err := workflow.ExecuteActivity(ctx, "CleanupActivity", activities.CleanupActivityInput{Path: cleanupPath}).Get(ctx, nil)
			if err != nil {
				logger.Error("Failed to execute cleanup activity", "error", err)
			}
		}
	}()

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
	// 3. Download emails from IMAP server
	////////////////////////////////////////
	DownloadActivityOutput := new(activities.DownloadActivityOutput)
	err = workflow.ExecuteActivity(
		ctx,
		"IMAPDownloadActivity",
		activities.DownloadActivityInput{
			Job: GetJobActivityOutput.Job,
		},
	).Get(ctx, DownloadActivityOutput)
	if err != nil {
		logger.Error("IMAP download failed", "error", err)
		return nil, err
	}
	cleanupPath = DownloadActivityOutput.FilePath // Set the path for cleanup

	logger.Info("Download completed", "filePath", DownloadActivityOutput.FilePath, "size", DownloadActivityOutput.Size)
	result.Status = true

	////////////////////////////////////////
	// 4. Request upload URL via API: POST /jobs/:job_id/backups/:backup_id/upload
	////////////////////////////////////////
	BackupUploadActivityOutput := new(activities.BackupUploadActivityOutput)
	err = workflow.ExecuteActivity(
		ctx,
		"BackupUploadActivity",
		activities.BackupUploadActivityInput{
			JobId:    input.JobId,
			BackupId: BackupRequestActivityOutput.ID.String(),
			FilePath: DownloadActivityOutput.FilePath,
			Size:     DownloadActivityOutput.Size,
			Checksum: DownloadActivityOutput.Checksum,
			Name:     DownloadActivityOutput.Name,
			MimeType: DownloadActivityOutput.MimeType,
		},
	).Get(ctx, BackupUploadActivityOutput)
	if err != nil {
		logger.Error("Failed to create backup upload activity", "error", err)
		return nil, err
	}

	////////////////////////////////////////
	// 5. Upload to S3 using presigned URL
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

	////////////////////////////////////////
	// 6. Confirm backup via API: POST /jobs/:job_id/backups/:backup_id/confirm
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
