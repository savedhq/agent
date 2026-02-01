package workflows

import (
	"agent/internal/temporal/activities"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// WebDAVBackupOutput defines the output for WebDAV backup workflow
type WebDAVBackupOutput struct {
	Status bool
}

// WebDAVBackupWorkflow performs WebDAV backup for agent jobs
func WebDAVBackupWorkflow(ctx workflow.Context, input GeneralWorkflowInput) (*WebDAVBackupOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("WebDAVBackupWorkflow started", "jobId", input.JobId)

	result := new(WebDAVBackupOutput)
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

	// 1. Get job details
	var getJobOutput activities.GetJobActivityOutput
	err := workflow.ExecuteActivity(ctx, "GetJobActivity", activities.GetJobActivityInput{JobId: input.JobId}).Get(ctx, &getJobOutput)
	if err != nil {
		logger.Error("Failed to get job details", "error", err)
		return nil, err
	}
	logger.Info("Job details fetched", "jobId", input.JobId)

	// 2. Create backup entity
	var backupRequestOutput activities.BackupRequestActivityOutput
	err = workflow.ExecuteActivity(ctx, "BackupRequestActivity", activities.BackupRequestActivityInput{Job: getJobOutput.Job}).Get(ctx, &backupRequestOutput)
	if err != nil {
		logger.Error("Failed to create backup request", "error", err)
		return nil, err
	}
	logger.Info("Backup created successfully", "backupID", backupRequestOutput.ID.String())

	// 3. Download file from WebDAV source
	var downloadOutput activities.DownloadWebDAVActivityOutput
	err = workflow.ExecuteActivity(ctx, "DownloadWebDAVActivity", activities.DownloadWebDAVActivityInput{Job: getJobOutput.Job}).Get(ctx, &downloadOutput)
	if err != nil {
		logger.Error("WebDAV download failed", "error", err)
		return nil, err
	}
	logger.Info("Download completed", "filePath", downloadOutput.FilePath, "size", downloadOutput.Size)
	result.Status = true

	defer func() {
		// Cleanup the downloaded file
		cleanupErr := workflow.ExecuteActivity(ctx, "CleanupActivity", activities.CleanupActivityInput{FilePath: downloadOutput.FilePath}).Get(ctx, nil)
		if cleanupErr != nil {
			logger.Error("Failed to cleanup file", "error", cleanupErr)
		}
	}()

	// 4. Request upload URL
	var backupUploadOutput activities.BackupUploadActivityOutput
	err = workflow.ExecuteActivity(ctx, "BackupUploadActivity", activities.BackupUploadActivityInput{
		JobId:    input.JobId,
		BackupId: backupRequestOutput.ID.String(),
		FilePath: downloadOutput.FilePath,
		Size:     downloadOutput.Size,
		Checksum: downloadOutput.Checksum,
		Name:     downloadOutput.Name,
		MimeType: downloadOutput.MimeType,
	}).Get(ctx, &backupUploadOutput)
	if err != nil {
		logger.Error("Failed to create backup upload activity", "error", err)
		return nil, err
	}

	// 5. Upload to S3
	var fileUploadOutput activities.FileUploadS3ActivityOutput
	err = workflow.ExecuteActivity(ctx, "FileUploadS3Activity", activities.FileUploadS3ActivityInput{
		FilePath:  downloadOutput.FilePath,
		UploadURL: backupUploadOutput.UploadURL,
		ExpiresAt: backupUploadOutput.ExpiresAt,
	}).Get(ctx, &fileUploadOutput)
	if err != nil {
		logger.Error("Failed to create file upload activity", "error", err)
		return nil, err
	}

	// 6. Confirm backup
	var backupConfirmOutput activities.BackupConfirmActivityOutput
	err = workflow.ExecuteActivity(ctx, "BackupConfirmActivity", activities.BackupConfirmActivityInput{
		JobId:    input.JobId,
		BackupId: backupRequestOutput.ID.String(),
		Status:   result.Status,
	}).Get(ctx, &backupConfirmOutput)
	if err != nil {
		logger.Error("Failed to create backup confirm activity", "error", err)
		return nil, err
	}

	return result, nil
}
