package workflows

import (
	"agent/internal"
	"agent/internal/temporal/activities"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func AWSDynamoDBBackupWorkflow(ctx workflow.Context, input GeneralWorkflowInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("AWSDynamoDBBackupWorkflow started", "jobId", input.JobId)

	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    5 * time.Minute,
			MaximumAttempts:    3,
		},
	})

	var getJobOut activities.GetJobActivityOutput
	if err := workflow.ExecuteActivity(ctx, internal.ActivityNameGetJob,
		activities.GetJobActivityInput{JobId: input.JobId}).Get(ctx, &getJobOut); err != nil {
		return err
	}

	var backupOut activities.BackupRequestActivityOutput
	if err := workflow.ExecuteActivity(ctx, internal.ActivityNameBackupRequest,
		activities.BackupRequestActivityInput{Job: getJobOut.Job}).Get(ctx, &backupOut); err != nil {
		return err
	}

	var dlOut activities.DownloadActivityOutput
	if err := workflow.ExecuteActivity(ctx, internal.ActivityNameAWSDynamoDBDump,
		activities.AWSDynamoDBDumpActivityInput{Job: getJobOut.Job}).Get(ctx, &dlOut); err != nil {
		return err
	}

	return ProcessAndUpload(ctx, getJobOut.Job, input.JobId, backupOut.ID.String(),
		dlOut.FilePath, dlOut.Size, dlOut.Checksum, dlOut.Name, dlOut.MimeType)
}
