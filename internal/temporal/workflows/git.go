package workflows

import (
	"agent/internal/temporal/activities"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// GitBackupOutput defines the output for Git backup workflow
type GitBackupOutput struct {
	Status bool
}

// GitBackupWorkflow performs Git backup for agent jobs
func GitBackupWorkflow(ctx workflow.Context, input GeneralWorkflowInput) (*GitBackupOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("GitBackupWorkflow started", "jobId", input.JobId)

	result := new(GitBackupOutput)
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

	var gitCloneOutput activities.GitCloneActivityOutput
	var compressOutput activities.FileCompressionActivityOutput

	defer func() {
		// Cleanup activity to remove temporary files
		cleanupInput := activities.CleanupActivityInput{
			Paths: []string{gitCloneOutput.Path, compressOutput.FilePath},
		}
		workflow.ExecuteActivity(ctx, "CleanupActivity", cleanupInput).Get(ctx, nil)
	}()

	// 1. Get job details
	var getJobOutput activities.GetJobActivityOutput
	err := workflow.ExecuteActivity(ctx, "GetJobActivity", activities.GetJobActivityInput{JobId: input.JobId}).Get(ctx, &getJobOutput)
	if err != nil {
		logger.Error("Failed to get job details", "error", err)
		return nil, err
	}

	// 2. Create backup entity
	var backupRequestOutput activities.BackupRequestActivityOutput
	err = workflow.ExecuteActivity(ctx, "BackupRequestActivity", activities.BackupRequestActivityInput{Job: getJobOutput.Job}).Get(ctx, &backupRequestOutput)
	if err != nil {
		logger.Error("Failed to create backup request", "error", err)
		return nil, err
	}

	// 3. Clone git repository
	err = workflow.ExecuteActivity(ctx, "GitCloneActivity", activities.GitCloneActivityInput{Job: getJobOutput.Job}).Get(ctx, &gitCloneOutput)
	if err != nil {
		logger.Error("Git clone failed", "error", err)
		return nil, err
	}

    // 4. Compress the cloned repository
	err = workflow.ExecuteActivity(ctx, "FileCompressionActivity", activities.FileCompressionActivityInput{FilePath: gitCloneOutput.Path}).Get(ctx, &compressOutput)
	if err != nil {
		logger.Error("File compression failed", "error", err)
		return nil, err
	}

	// 5. Request upload URL
	var backupUploadOutput activities.BackupUploadActivityOutput
	err = workflow.ExecuteActivity(ctx, "BackupUploadActivity", activities.BackupUploadActivityInput{
		JobId:    input.JobId,
		BackupId: backupRequestOutput.ID.String(),
		FilePath: compressOutput.FilePath,
		Size:     compressOutput.Size,
		Checksum: compressOutput.Checksum,
		Name:     compressOutput.Name,
		MimeType: "application/zip",
	}).Get(ctx, &backupUploadOutput)
	if err != nil {
		logger.Error("Failed to request upload URL", "error", err)
		return nil, err
	}

	// 6. Upload to S3
	var fileUploadOutput activities.FileUploadS3ActivityOutput
	err = workflow.ExecuteActivity(ctx, "FileUploadS3Activity", activities.FileUploadS3ActivityInput{
		FilePath:  compressOutput.FilePath,
		UploadURL: backupUploadOutput.UploadURL,
	}).Get(ctx, &fileUploadOutput)
	if err != nil {
		logger.Error("Failed to upload file", "error", err)
		return nil, err
	}

	// 7. Confirm backup
	err = workflow.ExecuteActivity(ctx, "BackupConfirmActivity", activities.BackupConfirmActivityInput{
		JobId:    input.JobId,
		BackupId: backupRequestOutput.ID.String(),
		Status:   true,
	}).Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to confirm backup", "error", err)
		return nil, err
	}

	result.Status = true
	return result, nil
}
