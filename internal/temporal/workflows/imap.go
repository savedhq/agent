package workflows

import (
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

	result := &IMAPBackupOutput{}

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

	var connectionID string
	err := workflow.ExecuteActivity(ctx, "IMAPConnectActivity").Get(ctx, &connectionID)
	if err != nil {
		logger.Error("IMAPConnectActivity failed", "error", err)
		return nil, err
	}
	defer func() {
		err := workflow.ExecuteActivity(ctx, "IMAPDisconnectActivity", connectionID).Get(ctx, nil)
		if err != nil {
			logger.Error("IMAPDisconnectActivity failed", "error", err)
		}
	}()

	var filePath string
	err = workflow.ExecuteActivity(ctx, "IMAPDownloadActivity", connectionID).Get(ctx, &filePath)
	if err != nil {
		logger.Error("IMAPDownloadActivity failed", "error", err)
		return nil, err
	}

	err = workflow.ExecuteActivity(ctx, "IMAPUploadActivity", filePath).Get(ctx, nil)
	if err != nil {
		logger.Error("IMAPUploadActivity failed", "error", err)
		return nil, err
	}

	result.Status = true

	return result, nil
}
