package workflows

import (
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

	// 1. Get job details
	getJobOutput := new(activities.GetJobActivityOutput)
	err := workflow.ExecuteActivity(ctx, "GetJobActivity",
		activities.GetJobActivityInput{JobId: input.JobId},
	).Get(ctx, getJobOutput)
	if err != nil {
		logger.Error("Failed to get job details", "error", err)
		return nil, err
	}

	// 2. Create backup request
	backupReqOutput := new(activities.BackupRequestActivityOutput)
	err = workflow.ExecuteActivity(ctx, "BackupRequestActivity",
		activities.BackupRequestActivityInput{Job: getJobOutput.Job},
	).Get(ctx, backupReqOutput)
	if err != nil {
		logger.Error("Failed to create backup request", "error", err)
		return nil, err
	}

	// 3. Download from S3
	downloadOutput := new(activities.S3DownloadActivityOutput)
	err = workflow.ExecuteActivity(ctx, "S3DownloadActivity",
		activities.S3DownloadActivityInput{Job: getJobOutput.Job},
	).Get(ctx, downloadOutput)
	if err != nil {
		logger.Error("S3 download failed", "error", err)
		return nil, err
	}

	// 4. Request upload URL
	uploadOutput := new(activities.BackupUploadActivityOutput)
	err = workflow.ExecuteActivity(ctx, "BackupUploadActivity",
		activities.BackupUploadActivityInput{
			JobId:    input.JobId,
			BackupId: backupReqOutput.ID.String(),
			FilePath: downloadOutput.FilePath,
			Size:     downloadOutput.Size,
			Checksum: downloadOutput.Checksum,
			Name:     downloadOutput.Name,
			MimeType: downloadOutput.MimeType,
		},
	).Get(ctx, uploadOutput)
	if err != nil {
		logger.Error("Failed to get upload URL", "error", err)
		return nil, err
	}

	// 5. Upload to S3 via presigned URL
	fileUploadOutput := new(activities.FileUploadS3ActivityOutput)
	err = workflow.ExecuteActivity(ctx, "FileUploadS3Activity",
		activities.FileUploadS3ActivityInput{
			FilePath:  downloadOutput.FilePath,
			UploadURL: uploadOutput.UploadURL,
			ExpiresAt: uploadOutput.ExpiresAt,
		},
	).Get(ctx, fileUploadOutput)
	if err != nil {
		logger.Error("Failed to upload file", "error", err)
		return nil, err
	}

	// 6. Confirm backup
	result.Status = true
	confirmOutput := new(activities.BackupConfirmActivityOutput)
	err = workflow.ExecuteActivity(ctx, "BackupConfirmActivity",
		activities.BackupConfirmActivityInput{
			JobId:    input.JobId,
			BackupId: backupReqOutput.ID.String(),
			Status:   result.Status,
		},
	).Get(ctx, confirmOutput)
	if err != nil {
		logger.Error("Failed to confirm backup", "error", err)
		return nil, err
	}

	return result, nil
}
