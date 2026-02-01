
package workflows

import (
	"agent/internal/temporal/activities"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ScriptBackupOutput defines the output for the script backup workflow
type ScriptBackupOutput struct {
	Status bool
}

// ScriptBackupWorkflow performs a script-based backup
func ScriptBackupWorkflow(ctx workflow.Context, input GeneralWorkflowInput) (*ScriptBackupOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("ScriptBackupWorkflow started", "jobId", input.JobId)

	result := new(ScriptBackupOutput)
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

	// 2. Create backup entity via API
	var backupRequestOutput activities.BackupRequestActivityOutput
	err = workflow.ExecuteActivity(ctx, "BackupRequestActivity", activities.BackupRequestActivityInput{Job: getJobOutput.Job}).Get(ctx, &backupRequestOutput)
	if err != nil {
		logger.Error("Failed to create backup request", "error", err)
		return nil, err
	}
	logger.Info("Backup created successfully", "backupID", backupRequestOutput.ID.String())

	// 3. Execute the script
	var executeScriptOutput activities.ExecuteScriptActivityOutput
	err = workflow.ExecuteActivity(ctx, "ExecuteScriptActivity", activities.ExecuteScriptActivityInput{Job: getJobOutput.Job}).Get(ctx, &executeScriptOutput)
	if err != nil {
		logger.Error("Failed to execute script", "error", err)
		return nil, err
	}
	logger.Info("Script executed successfully", "outputFile", executeScriptOutput.OutputFile)

	// 4. Request upload URL
	var backupUploadOutput activities.BackupUploadActivityOutput
	err = workflow.ExecuteActivity(ctx, "BackupUploadActivity", activities.BackupUploadActivityInput{
		JobId:    input.JobId,
		BackupId: backupRequestOutput.ID.String(),
		FilePath: executeScriptOutput.OutputFile,
		Size:     executeScriptOutput.Size,
		Checksum: executeScriptOutput.Checksum,
		Name:     executeScriptOutput.Name,
		MimeType: "application/octet-stream",
	}).Get(ctx, &backupUploadOutput)
	if err != nil {
		logger.Error("Failed to get upload URL", "error", err)
		return nil, err
	}

	// 5. Upload to S3
	var fileUploadOutput activities.FileUploadS3ActivityOutput
	err = workflow.ExecuteActivity(ctx, "FileUploadS3Activity", activities.FileUploadS3ActivityInput{
		FilePath:  executeScriptOutput.OutputFile,
		UploadURL: backupUploadOutput.UploadURL,
	}).Get(ctx, &fileUploadOutput)
	if err != nil {
		logger.Error("Failed to upload file", "error", err)
		return nil, err
	}
	logger.Info("File uploaded successfully")
    result.Status = true

	// 6. Confirm backup
	var backupConfirmOutput activities.BackupConfirmActivityOutput
	err = workflow.ExecuteActivity(ctx, "BackupConfirmActivity", activities.BackupConfirmActivityInput{
		JobId:    input.JobId,
		BackupId: backupRequestOutput.ID.String(),
		Status:   result.Status,
	}).Get(ctx, &backupConfirmOutput)
	if err != nil {
		logger.Error("Failed to confirm backup", "error", err)
		return nil, err
	}

	logger.Info("ScriptBackupWorkflow completed", "jobId", input.JobId)
	return result, nil
}
