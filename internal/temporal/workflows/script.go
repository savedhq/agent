package workflows

import (
	"agent/internal"
	"agent/internal/temporal/activities"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func ScriptBackupWorkflow(ctx workflow.Context, input GeneralWorkflowInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("ScriptBackupWorkflow started", "jobId", input.JobId)

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

	// 1. Get job from config
	var getJobOut activities.GetJobActivityOutput
	err := workflow.ExecuteActivity(ctx, internal.ActivityNameGetJob,
		activities.GetJobActivityInput{JobId: input.JobId},
	).Get(ctx, &getJobOut)
	if err != nil {
		return err
	}

	// 2. Create backup entity via API
	var backupOut activities.BackupRequestActivityOutput
	err = workflow.ExecuteActivity(ctx, internal.ActivityNameBackupRequest,
		activities.BackupRequestActivityInput{Job: getJobOut.Job},
	).Get(ctx, &backupOut)
	if err != nil {
		return err
	}

	// 3. Run script to generate backup file
	var scriptOut activities.ScriptRunActivityOutput
	err = workflow.ExecuteActivity(ctx, internal.ActivityNameScriptRun,
		activities.ScriptRunActivityInput{Job: getJobOut.Job},
	).Get(ctx, &scriptOut)
	if err != nil {
		return err
	}

	// 4-7. Compress → Encrypt → Upload → Confirm
	return ProcessAndUpload(
		ctx,
		getJobOut.Job,
		input.JobId,
		backupOut.ID.String(),
		scriptOut.FilePath,
		scriptOut.Size,
		scriptOut.Checksum,
		scriptOut.Name,
		scriptOut.MimeType,
	)
}
