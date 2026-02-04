package workflows

import (
	"agent/internal/config/job"
	"agent/internal/temporal/activities"
	"agent/pkg/names"
	"fmt"
	"path/filepath"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type MySQLBackupOutput struct {
	Status bool
}

func MySQLBackupWorkflow(ctx workflow.Context, input GeneralWorkflowInput) (*MySQLBackupOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("MySQLBackupWorkflow started", "jobId", input.JobId)
	logger.Info("MySQLBackupWorkflow started", "Provider", input.Provider)

	result := new(MySQLBackupOutput)
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

	var tempDir string

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
	// 3. Create temp directory for dump
	////////////////////////////////////////
	CreateTempDirActivityOutput := new(activities.CreateTempDirOutput)
	err = workflow.ExecuteActivity(
		ctx,
		names.ActivityNameCreateTempDir,
		activities.CreateTempDirInput{Pattern: "mysql-backup-"},
	).Get(ctx, CreateTempDirActivityOutput)
	if err != nil {
		logger.Error("Failed to create temp directory", "error", err)
		return nil, err
	}
	tempDir = CreateTempDirActivityOutput.Path

	////////////////////////////////////////
	// 4. Run MySQL dump
	////////////////////////////////////////
	mysqlConfig, ok := GetJobActivityOutput.Job.Config.(*job.MySQLJobConfig)
	if !ok {
		logger.Error("Failed to assert mysql job config")
		return nil, fmt.Errorf("failed to assert job config to MySQLJobConfig")
	}

	MySQLDumpActivityOutput := new(activities.MySQLDumpOutput)
	err = workflow.ExecuteActivity(
		ctx,
		names.ActivityNameMySQLDump,
		activities.MySQLDumpInput{
			ConnectionString: mysqlConfig.ConnectionString,
			OutputPath:       filepath.Join(tempDir, BackupRequestActivityOutput.ID.String()),
		},
	).Get(ctx, MySQLDumpActivityOutput)
	if err != nil {
		logger.Error("MySQL dump failed", "error", err)
		return nil, err
	}

	filePath := MySQLDumpActivityOutput.Path
	logger.Info("Dump completed", "filePath", filePath)
	result.Status = true

	////////////////////////////////////////
	// 5. Compress file (if enabled)
	////////////////////////////////////////
	if GetJobActivityOutput.Job.Compression.Enabled {
		FileCompressionActivityOutput := new(activities.FileCompressionActivityOutput)
		err = workflow.ExecuteActivity(
			ctx,
			names.ActivityNameCompressFile,
			activities.FileCompressionActivityInput{
				FilePath: filePath,
				Provider: GetJobActivityOutput.Job.Compression.Algorithm,
			},
		).Get(ctx, FileCompressionActivityOutput)
		if err != nil {
			logger.Error("Failed to compress file", "error", err)
			return nil, err
		}
		filePath = FileCompressionActivityOutput.FilePath
		logger.Info("Compression completed", "filePath", filePath)
	}

	////////////////////////////////////////
	// 6. Encrypt file (if enabled)
	////////////////////////////////////////
	if GetJobActivityOutput.Job.Encryption.Enabled {
		FileEncryptionActivityOutput := new(activities.FileEncryptionActivityOutput)
		err = workflow.ExecuteActivity(
			ctx,
			names.ActivityNameEncryptFile,
			activities.FileEncryptionActivityInput{
				FilePath: filePath,
				Provider: GetJobActivityOutput.Job.Encryption.Algorithm,
				Key:      GetJobActivityOutput.Job.Encryption.PublicKey,
			},
		).Get(ctx, FileEncryptionActivityOutput)
		if err != nil {
			logger.Error("Failed to encrypt file", "error", err)
			return nil, err
		}
		filePath = FileEncryptionActivityOutput.FilePath
		logger.Info("Encryption completed", "filePath", filePath)
	}

	////////////////////////////////////////
	// 7. Request upload URL via API: POST /jobs/:job_id/backups/:backup_id/upload
	////////////////////////////////////////
	BackupUploadActivityOutput := new(activities.BackupUploadActivityOutput)
	err = workflow.ExecuteActivity(
		ctx,
		names.ActivityNameBackupUpload,
		activities.BackupUploadActivityInput{
			JobId:    input.JobId,
			BackupId: BackupRequestActivityOutput.ID.String(),
			FilePath: filePath,
		},
	).Get(ctx, BackupUploadActivityOutput)
	if err != nil {
		logger.Error("Failed to create backup upload activity", "error", err)
		return nil, err
	}

	////////////////////////////////////////
	// 8. Upload to S3 using presigned URL
	////////////////////////////////////////
	FileUploadS3ActivityOutput := new(activities.FileUploadS3ActivityOutput)
	err = workflow.ExecuteActivity(
		ctx,
		names.ActivityNameFileUploadS3,
		activities.FileUploadS3ActivityInput{
			FilePath:  filePath,
			UploadURL: BackupUploadActivityOutput.UploadURL,
			ExpiresAt: BackupUploadActivityOutput.ExpiresAt,
		},
	).Get(ctx, FileUploadS3ActivityOutput)
	if err != nil {
		logger.Error("Failed to create file upload activity", "error", err)
		return nil, err
	}

	////////////////////////////////////////
	// 9. Confirm backup via API: POST /jobs/:job_id/backups/:backup_id/confirm
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
