package workflows

import (
	"fmt"
	"path/filepath"
	"time"

	"agent/internal/config/job"
	"agent/internal/temporal/activities"
	"agent/pkg/names"
	"go.temporal.io/sdk/workflow"
)

func MySQLBackupWorkflow(ctx workflow.Context, input GeneralWorkflowInput) error {
	var err error
	logger := workflow.GetLogger(ctx)

	// Keep track of the file path throughout the workflow
	var filePath string
	var tempDir string

	// Defer cleanup actions to run at the end of the workflow
	defer func() {
		if tempDir != "" {
			var result activities.RemoveFileOutput
			err = workflow.ExecuteActivity(ctx, "RemoveFileActivity", activities.RemoveFileInput{Path: tempDir}).Get(ctx, &result)
			if err != nil {
				logger.Error("failed to cleanup temp directory", "path", tempDir, "error", err)
			}
		}
	}()

	// Define activity options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 1 * time.Hour,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// 1. Get job config
	var jobConfig job.Job
	err = workflow.ExecuteActivity(ctx, names.ActivityNameGetJob, activities.GetJobActivityInput{JobId: input.JobId}).Get(ctx, &jobConfig)
	if err != nil {
		return fmt.Errorf("failed to get job config: %w", err)
	}

	// 2. Request backup from backend
	var backupReq activities.BackupRequestActivityOutput
	err = workflow.ExecuteActivity(ctx, names.ActivityNameBackupRequest, activities.BackupRequestActivityInput{Job: &jobConfig}).Get(ctx, &backupReq)
	if err != nil {
		return fmt.Errorf("failed to request backup: %w", err)
	}
	logger.Info("Backup requested", "backup_id", backupReq.ID)

	// Create a temporary directory for the backup
	var tempDirOutput activities.CreateTempDirOutput
	err = workflow.ExecuteActivity(ctx, "CreateTempDirActivity", activities.CreateTempDirInput{Pattern: "mysql-backup-"}).Get(ctx, &tempDirOutput)
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	tempDir = tempDirOutput.Path

	// 3. Run dump
	var dumpOutput activities.MySQLDumpOutput
	mysqlConfig, ok := jobConfig.Config.(*job.MySQLJobConfig)
	if !ok {
		return fmt.Errorf("failed to assert mysql job config")
	}
	err = workflow.ExecuteActivity(ctx, names.ActivityNameMySQLDump, activities.MySQLDumpInput{
		ConnectionString: mysqlConfig.ConnectionString,
		OutputPath:       filepath.Join(tempDir, backupReq.ID.String()),
	}).Get(ctx, &dumpOutput)
	if err != nil {
		return fmt.Errorf("failed to run mysql dump: %w", err)
	}
	filePath = dumpOutput.Path

	// 4. Compress (optional)
	if jobConfig.Compression.Enabled {
		var compOutput activities.FileCompressionActivityOutput
		err = workflow.ExecuteActivity(ctx, names.ActivityNameCompressFile, activities.FileCompressionActivityInput{
			InputPath:  filePath,
			OutputPath: filePath + ".gz",
		}).Get(ctx, &compOutput)
		if err != nil {
			return fmt.Errorf("failed to compress file: %w", err)
		}
		filePath = compOutput.OutputPath
	}

	// 5. Encrypt (optional)
	if jobConfig.Encryption.Enabled {
		var encOutput activities.FileEncryptionActivityOutput
		err = workflow.ExecuteActivity(ctx, names.ActivityNameEncryptFile, activities.FileEncryptionActivityInput{
			InputPath:  filePath,
			OutputPath: filePath + ".enc",
		}).Get(ctx, &encOutput)
		if err != nil {
			return fmt.Errorf("failed to encrypt file: %w", err)
		}
		filePath = encOutput.OutputPath
	}

	// 6. Get presigned upload URL
	var uploadURL activities.BackupUploadActivityOutput
	err = workflow.ExecuteActivity(ctx, names.ActivityNameBackupUpload, backupReq.ID).Get(ctx, &uploadURL)
	if err != nil {
		return fmt.Errorf("failed to get presigned upload url: %w", err)
	}

	// 7. Upload to S3
	err = workflow.ExecuteActivity(ctx, names.ActivityNameFileUploadS3, activities.FileUploadS3ActivityInput{
		FilePath:  filePath,
		UploadURL: uploadURL.UploadURL,
	}).Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to upload file to s3: %w", err)
	}

	// 8. Confirm backup
	err = workflow.ExecuteActivity(ctx, names.ActivityNameBackupConfirm, backupReq.ID).Get(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to confirm backup: %w", err)
	}

	logger.Info("MySQL backup workflow completed successfully")
	return nil
}
