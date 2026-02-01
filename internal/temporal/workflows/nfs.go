package workflows

import (
	"agent/internal/config/job"
	"agent/internal/temporal/activities"
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// NFSBackupOutput defines the output for NFS backup workflow
type NFSBackupOutput struct {
	Status bool
}

// NFSBackupWorkflow performs NFS backup for agent jobs
func NFSBackupWorkflow(ctx workflow.Context, input GeneralWorkflowInput) (*NFSBackupOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("NFSBackupWorkflow started", "jobId", input.JobId)

	result := new(NFSBackupOutput)
	// Activity options
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
	getJobOutput := new(activities.GetJobActivityOutput)
	err := workflow.ExecuteActivity(ctx, "GetJobActivity", activities.GetJobActivityInput{JobId: input.JobId}).Get(ctx, getJobOutput)
	if err != nil {
		logger.Error("Failed to get job details", "error", err)
		return nil, err
	}

	nfsConfig, ok := getJobOutput.Job.Config.(*job.NFSConfig)
	if !ok {
		logger.Error("Failed to cast job config to NFSConfig")
		return nil, temporal.NewNonRetryableApplicationError("invalid job config type", "InvalidConfig", nil)
	}

	// 2. Mount NFS share - with deferred unmount
	mountInput := activities.MountActivityInput{
		RemotePath: nfsConfig.RemotePath,
		LocalPath:  nfsConfig.LocalPath,
	}
	err = workflow.ExecuteActivity(ctx, "MountActivity", mountInput).Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to mount NFS share", "error", err)
		return nil, err
	}

	defer func() {
		// Use a disconnected context to ensure unmount runs even if the workflow is cancelled.
		disconnectedCtx, _ := workflow.NewDisconnectedContext(ctx)
		unmountInput := activities.UnmountActivityInput{LocalPath: nfsConfig.LocalPath}
		err := workflow.ExecuteActivity(disconnectedCtx, "UnmountActivity", unmountInput).Get(disconnectedCtx, nil)
		if err != nil {
			logger.Error("Failed to unmount NFS share in cleanup", "error", err)
		} else {
			logger.Info("NFS share unmounted successfully in cleanup")
		}
	}()

	logger.Info("NFS share mounted successfully", "localPath", nfsConfig.LocalPath)

	// 3. Create backup entity via API
	backupRequestOutput := new(activities.BackupRequestActivityOutput)
	err = workflow.ExecuteActivity(ctx, "BackupRequestActivity", activities.BackupRequestActivityInput{Job: getJobOutput.Job}).Get(ctx, backupRequestOutput)
	if err != nil {
		logger.Error("Failed to create backup request", "error", err)
		return nil, err
	}
	logger.Info("Backup created successfully", "backupID", backupRequestOutput.ID.String())

	// 4. Compress the mounted directory
	zipActivityOutput := new(activities.ZipActivityOutput)
	zipActivityInput := activities.ZipActivityInput{
		SourcePath:      nfsConfig.LocalPath,
		DestinationPath: fmt.Sprintf("/tmp/nfs-backup-%s.zip", workflow.GetInfo(ctx).WorkflowExecution.ID),
	}
	err = workflow.ExecuteActivity(ctx, "ZipActivity", zipActivityInput).Get(ctx, zipActivityOutput)
	if err != nil {
		logger.Error("Failed to zip directory", "error", err)
		return nil, err
	}
	logger.Info("Directory zipped successfully", "path", zipActivityOutput.FilePath)

	defer func() {
		// Use a disconnected context to ensure cleanup runs even if the workflow is cancelled.
		disconnectedCtx, _ := workflow.NewDisconnectedContext(ctx)
		cleanupInput := activities.CleanupActivityInput{FilePath: zipActivityOutput.FilePath}
		err := workflow.ExecuteActivity(disconnectedCtx, "CleanupActivity", cleanupInput).Get(disconnectedCtx, nil)
		if err != nil {
			logger.Error("Failed to cleanup temporary zip file", "error", err)
		} else {
			logger.Info("Temporary zip file cleaned up successfully")
		}
	}()

	// 5. Request upload URL
	backupUploadOutput := new(activities.BackupUploadActivityOutput)
	uploadActivityInput := activities.BackupUploadActivityInput{
		JobId:    input.JobId,
		BackupId: backupRequestOutput.ID.String(),
		FilePath: zipActivityOutput.FilePath,
		Size:     zipActivityOutput.Size,
		Checksum: zipActivityOutput.Checksum,
		Name:     zipActivityOutput.Name,
		MimeType: zipActivityOutput.MimeType,
	}
	err = workflow.ExecuteActivity(ctx, "BackupUploadActivity", uploadActivityInput).Get(ctx, backupUploadOutput)
	if err != nil {
		logger.Error("Failed to get upload URL", "error", err)
		return nil, err
	}

	// 6. Upload to S3
	uploadS3Input := activities.FileUploadS3ActivityInput{
		FilePath:  zipActivityOutput.FilePath,
		UploadURL: backupUploadOutput.UploadURL,
		ExpiresAt: backupUploadOutput.ExpiresAt,
	}
	err = workflow.ExecuteActivity(ctx, "FileUploadS3Activity", uploadS3Input).Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to upload file to S3", "error", err)
		return nil, err
	}
	logger.Info("File uploaded to S3 successfully")
	result.Status = true

	// 7. Confirm backup
	confirmInput := activities.BackupConfirmActivityInput{
		JobId:    input.JobId,
		BackupId: backupRequestOutput.ID.String(),
		Status:   result.Status,
	}
	err = workflow.ExecuteActivity(ctx, "BackupConfirmActivity", confirmInput).Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to confirm backup", "error", err)
		return nil, err
	}
	logger.Info("Backup confirmed successfully")

	return result, nil
}
