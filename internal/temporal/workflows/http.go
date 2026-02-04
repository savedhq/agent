package workflows

import (
	"agent/internal/temporal/activities"
	"agent/pkg/names"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// HTTPBackupOutput defines the output for HTTP backup workflow
type HTTPBackupOutput struct {
	Status bool
}

// HTTPBackupWorkflow performs HTTP backup for agent jobs
// Agent runs on client systems and uses REST API to communicate with backend
func HTTPBackupWorkflow(ctx workflow.Context, input GeneralWorkflowInput) (*HTTPBackupOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("HTTPBackupWorkflow started", "jobId", input.JobId)
	logger.Info("HTTPBackupWorkflow started", "Provider", input.Provider)

	result := new(HTTPBackupOutput)
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

	////////////////////////////////////////
	// 1. Get job details from config.yaml
	////////////////////////////////////////
	GetJobActivityOutput := new(activities.GetJobActivityOutput)

	activity := workflow.ExecuteActivity(
		ctx,
		names.ActivityNameGetJob,
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
		names.ActivityNameBackupRequest,
		activities.BackupRequestActivityInput{Job: GetJobActivityOutput.Job},
	).Get(ctx, BackupRequestActivityOutput)
	if err != nil {
		logger.Error("Failed to create backup request", "error", err)
		return nil, err
	}

	logger.Info("Backup created successfully", "backupID", BackupRequestActivityOutput.ID.String())

	////////////////////////////////////////
	// 3. Download file from HTTP source
	////////////////////////////////////////
	DownloadActivityOutput := new(activities.DownloadActivityOutput)
	err = workflow.ExecuteActivity(
		ctx,
		names.ActivityNameDownload,
		activities.DownloadActivityInput{
			Job: GetJobActivityOutput.Job,
		},
	).Get(ctx, DownloadActivityOutput)
	if err != nil {
		logger.Error("HTTP download failed", "error", err)
		return nil, err
	}

	logger.Info("Download completed", "filePath", DownloadActivityOutput.FilePath, "size", DownloadActivityOutput.Size)
	result.Status = true
	////////////////////////////////////////
	// 4. Compress + encrypt file
	////////////////////////////////////////
	var compressedFilePath string
	if GetJobActivityOutput.Job.Compression.Enabled {
		var compressionOutput activities.FileCompressionActivityOutput
		err = workflow.ExecuteActivity(ctx, names.ActivityNameCompressFile,
			activities.FileCompressionActivityInput{FilePath: DownloadActivityOutput.FilePath, Provider: GetJobActivityOutput.Job.Compression.Algorithm}).Get(ctx, &compressionOutput)
		if err != nil {
			return nil, err
		}
		compressedFilePath = compressionOutput.FilePath
		workflow.ExecuteActivity(ctx, names.ActivityNameFileCleanup, activities.FileCleanupActivityInput{FilePath: DownloadActivityOutput.FilePath}).Get(ctx, nil)
	} else {
		compressedFilePath = DownloadActivityOutput.FilePath
	}

	var encryptedFilePath string
	if GetJobActivityOutput.Job.Encryption.Enabled {
		var encryptionOutput activities.FileEncryptionActivityOutput
		err = workflow.ExecuteActivity(ctx, names.ActivityNameEncryptFile,
			activities.FileEncryptionActivityInput{FilePath: compressedFilePath, Provider: GetJobActivityOutput.Job.Encryption.Algorithm, Key: GetJobActivityOutput.Job.Encryption.PublicKey}).Get(ctx, &encryptionOutput)
		if err != nil {
			return nil, err
		}
		encryptedFilePath = encryptionOutput.FilePath
		workflow.ExecuteActivity(ctx, names.ActivityNameFileCleanup, activities.FileCleanupActivityInput{FilePath: compressedFilePath}).Get(ctx, nil)
	} else {
		encryptedFilePath = compressedFilePath
	}

	////////////////////////////////////////
	// 5. Request upload URL via API: POST /jobs/:job_id/backups/:backup_id/upload
	////////////////////////////////////////
	BackupUploadActivityOutput := new(activities.BackupUploadActivityOutput)
	err = workflow.ExecuteActivity(
		ctx,
		names.ActivityNameBackupUpload,
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
	// 6. Upload to S3 using presigned URL
	////////////////////////////////////////
	FileUploadS3ActivityOutput := new(activities.FileUploadS3ActivityOutput)
	err = workflow.ExecuteActivity(
		ctx,
		names.ActivityNameFileUploadS3,
		activities.FileUploadS3ActivityInput{
			FilePath:  encryptedFilePath,
			UploadURL: BackupUploadActivityOutput.UploadURL,
			ExpiresAt: BackupUploadActivityOutput.ExpiresAt,
		},
	).Get(ctx, FileUploadS3ActivityOutput)
	workflow.ExecuteActivity(ctx, names.ActivityNameFileCleanup, activities.FileCleanupActivityInput{FilePath: encryptedFilePath}).Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to create file upload activity", "error", err)
		return nil, err
	}

	////////////////////////////////////////
	// 7. Confirm backup via API: POST /jobs/:job_id/backups/:backup_id/confirm
	////////////////////////////////////////
	BackupConfirmActivityOutput := new(activities.BackupConfirmActivityOutput)
	err = workflow.ExecuteActivity(
		ctx,
		names.ActivityNameBackupConfirm,
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
