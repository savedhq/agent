package workflows

import (
	"agent/internal/temporal/activities"
	"agent/pkg/names"
	"time"

	"go.temporal.io/sdk/workflow"
)

import (
	"agent/internal/config/job"
	"fmt"
)

func PostgreSQLBackupWorkflow(ctx workflow.Context, input activities.GetJobActivityInput) error {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 1 * time.Hour,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var jobDetails activities.GetJobActivityOutput
	err := workflow.ExecuteActivity(ctx, names.ActivityNameGetJob, input).Get(ctx, &jobDetails)
	if err != nil {
		return err
	}

	var backupRequestResponse activities.BackupRequestActivityOutput
	err = workflow.ExecuteActivity(ctx, names.ActivityNameBackupRequest,
		activities.BackupRequestActivityInput{Job: jobDetails.Job}).Get(ctx, &backupRequestResponse)
	if err != nil {
		return err
	}

	// Type assert the generic JobConfig to our specific PostgreSQLJobConfig
	pgConfig, ok := jobDetails.Job.Config.(*job.PostgreSQLJobConfig)
	if !ok {
		return fmt.Errorf("failed to assert job config to PostgreSQLJobConfig")
	}

	var dumpFilePath string
	err = workflow.ExecuteActivity(ctx, names.ActivityNamePostgreSQLDump, pgConfig).Get(ctx, &dumpFilePath)
	if err != nil {
		return err
	}

	var compressedFilePath string
	if jobDetails.Job.Compression.Enabled {
		var compressionOutput activities.FileCompressionActivityOutput
		err = workflow.ExecuteActivity(ctx, names.ActivityNameCompressFile,
			activities.FileCompressionActivityInput{FilePath: dumpFilePath, Provider: jobDetails.Job.Compression.Algorithm}).Get(ctx, &compressionOutput)
		if err != nil {
			return err
		}
		compressedFilePath = compressionOutput.FilePath
		// Cleanup the dump file
		workflow.ExecuteActivity(ctx, names.ActivityNameFileCleanup, activities.FileCleanupActivityInput{FilePath: dumpFilePath}).Get(ctx, nil)
	} else {
		compressedFilePath = dumpFilePath
	}

	var encryptedFilePath string
	if jobDetails.Job.Encryption.Enabled {
		var encryptionOutput activities.FileEncryptionActivityOutput
		err = workflow.ExecuteActivity(ctx, names.ActivityNameEncryptFile,
			activities.FileEncryptionActivityInput{FilePath: compressedFilePath, Provider: jobDetails.Job.Encryption.Algorithm, Key: jobDetails.Job.Encryption.PublicKey}).Get(ctx, &encryptionOutput)
		if err != nil {
			return err
		}
		encryptedFilePath = encryptionOutput.FilePath
		// Cleanup the compressed file
		workflow.ExecuteActivity(ctx, names.ActivityNameFileCleanup, activities.FileCleanupActivityInput{FilePath: compressedFilePath}).Get(ctx, nil)
	} else {
		encryptedFilePath = compressedFilePath
	}

	var uploadURL activities.BackupUploadActivityOutput
	err = workflow.ExecuteActivity(ctx, names.ActivityNameBackupUpload,
		activities.BackupUploadActivityInput{JobId: jobDetails.Job.ID, BackupId: backupRequestResponse.ID.String()}).Get(ctx, &uploadURL)
	if err != nil {
		return err
	}

	err = workflow.ExecuteActivity(ctx, names.ActivityNameFileUploadS3,
		activities.FileUploadS3ActivityInput{UploadURL: uploadURL.UploadURL, FilePath: encryptedFilePath}).Get(ctx, nil)
	defer workflow.ExecuteActivity(ctx, names.ActivityNameFileCleanup, activities.FileCleanupActivityInput{FilePath: encryptedFilePath}).Get(ctx, nil)
	if err != nil {
		return err
	}

	err = workflow.ExecuteActivity(ctx, names.ActivityNameBackupConfirm,
		activities.BackupConfirmActivityInput{JobId: jobDetails.Job.ID, BackupId: backupRequestResponse.ID.String(), Status: true}).Get(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}
