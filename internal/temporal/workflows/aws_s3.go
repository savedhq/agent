package workflows

import (
	"agent/internal/config/job"
	"agent/internal/temporal/activities"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type AWSS3BackupWorkflowOutput struct {
	Status bool
}

func AWSS3BackupWorkflow(ctx workflow.Context, input GeneralWorkflowInput) (*AWSS3BackupWorkflowOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("AWSS3BackupWorkflow started", "jobId", input.JobId)

	result := new(AWSS3BackupWorkflowOutput)
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

	s3Config, ok := GetJobActivityOutput.Job.Config.(*job.AWSS3Config)
	if !ok {
		err := temporal.NewApplicationError("invalid job config type", "INVALID_CONFIG_TYPE")
		logger.Error("Failed to cast job config to AWSS3Config", "error", err)
		return nil, err
	}

	SyncS3BucketActivityOutput := new(activities.SyncS3BucketActivityOutput)
	err := workflow.ExecuteActivity(
		ctx,
		"SyncS3BucketActivity",
		activities.SyncS3BucketActivityInput{
			SourceRegion:          s3Config.Source.Region,
			SourceBucket:          s3Config.Source.Bucket,
			SourcePath:            s3Config.Source.Path,
			SourceAccessKeyID:     s3Config.Source.AccessKeyID,
			SourceSecretAccessKey: s3Config.Source.SecretAccessKey,
			SourceEndpoint:        s3Config.Source.Endpoint,
			DestRegion:          s3Config.Destination.Region,
			DestBucket:          s3Config.Destination.Bucket,
			DestPath:            s3Config.Destination.Path,
			DestAccessKeyID:     s3Config.Destination.AccessKeyID,
			DestSecretAccessKey: s3Config.Destination.SecretAccessKey,
			DestEndpoint:        s3Config.Destination.Endpoint,
		},
	).Get(ctx, SyncS3BucketActivityOutput)
	if err != nil {
		logger.Error("Failed to sync S3 bucket", "error", err)
		return nil, err
	}

	result.Status = true
	return result, nil
}
