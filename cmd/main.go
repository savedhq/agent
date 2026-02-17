package main

import (
	names "agent/internal"
	"agent/internal/authentication"
	"agent/internal/config"
	"agent/internal/hub"
	"agent/internal/temporal/activities"
	"agent/internal/temporal/workflows"
	"context"
	"crypto/tls"
	"log"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/contrib/envconfig"
	"go.temporal.io/sdk/workflow"

	temporalclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {

	ctx := context.Background()

	// Load config from file
	cfg, err := config.NewConfig(ctx, "")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	authService := authentication.NewAuth0Service(&cfg.Auth)

	// Fetch hub configuration from API
	hubConfig, err := hub.LoadHubConfig(ctx, authService, cfg)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	clientOptions := envconfig.MustLoadDefaultClientOptions()

	if hubConfig.TLS.Enabled {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
			NextProtos: []string{"h2"},
		}

		clientOptions.ConnectionOptions = temporalclient.ConnectionOptions{
			TLS: tlsConfig,
		}
	}

	clientOptions.HostPort = hubConfig.Server
	clientOptions.Namespace = hubConfig.Workspace

	clientOptions.Credentials = temporalclient.NewAPIKeyDynamicCredentials(func(ctx context.Context) (string, error) {
		return authService.Token()
	})

	c, err := temporalclient.DialContext(ctx, clientOptions)

	if err != nil {
		log.Fatalln("Unable to create Temporal client", err)
	}
	log.Printf("Connected to workspace: %s, queue: %s", hubConfig.Workspace, hubConfig.Queue)

	defer c.Close()

	workerOptions := worker.Options{
		EnableSessionWorker: true,
	}

	// Create worker
	w := worker.New(c, hubConfig.Queue, workerOptions)

	// Register workflows
	w.RegisterWorkflowWithOptions(workflows.HTTPBackupWorkflow, workflow.RegisterOptions{Name: names.WorkflowNameHTTP})
	w.RegisterWorkflowWithOptions(workflows.FTPBackupWorkflow, workflow.RegisterOptions{Name: names.WorkflowNameFTP})
	w.RegisterWorkflowWithOptions(workflows.WebDAVBackupWorkflow, workflow.RegisterOptions{Name: names.WorkflowNameWebDAV})
	w.RegisterWorkflowWithOptions(workflows.GitBackupWorkflow, workflow.RegisterOptions{Name: names.WorkflowNameGit})
	w.RegisterWorkflowWithOptions(workflows.MySQLBackupWorkflow, workflow.RegisterOptions{Name: names.WorkflowNameMySQL})
	w.RegisterWorkflowWithOptions(workflows.PostgreSQLBackupWorkflow, workflow.RegisterOptions{Name: names.WorkflowNamePostgreSQL})
	w.RegisterWorkflowWithOptions(workflows.MSSQLBackupWorkflow, workflow.RegisterOptions{Name: names.WorkflowNameMSSQL})
	w.RegisterWorkflowWithOptions(workflows.RedisBackupWorkflow, workflow.RegisterOptions{Name: names.WorkflowNameRedis})
	w.RegisterWorkflowWithOptions(workflows.AWSS3BackupWorkflow, workflow.RegisterOptions{Name: names.WorkflowNameAWSS3})
	w.RegisterWorkflowWithOptions(workflows.AWSDynamoDBBackupWorkflow, workflow.RegisterOptions{Name: names.WorkflowNameAWSDynamoDB})
	w.RegisterWorkflowWithOptions(workflows.ScriptBackupWorkflow, workflow.RegisterOptions{Name: names.WorkflowNameScript})

	// Create activities instance with dependency injection
	acts := activities.NewActivities(cfg, authService, *hubConfig, c)

	// Register shared activities
	w.RegisterActivityWithOptions(acts.BackupRequestActivity, activity.RegisterOptions{Name: names.ActivityNameBackupRequest})
	w.RegisterActivityWithOptions(acts.BackupUploadActivity, activity.RegisterOptions{Name: names.ActivityNameBackupUpload})
	w.RegisterActivityWithOptions(acts.BackupConfirmActivity, activity.RegisterOptions{Name: names.ActivityNameBackupConfirm})
	w.RegisterActivityWithOptions(acts.FileCompressionActivity, activity.RegisterOptions{Name: names.ActivityNameCompressFile})
	w.RegisterActivityWithOptions(acts.FileEncryptionActivity, activity.RegisterOptions{Name: names.ActivityNameEncryptFile})
	w.RegisterActivityWithOptions(acts.GetJobActivity, activity.RegisterOptions{Name: names.ActivityNameGetJob})
	w.RegisterActivityWithOptions(acts.FileUploadS3Activity, activity.RegisterOptions{Name: names.ActivityNameFileUploadS3})
	w.RegisterActivityWithOptions(acts.FileCleanupActivity, activity.RegisterOptions{Name: names.ActivityNameFileCleanup})
	w.RegisterActivityWithOptions(acts.CreateTempDirActivity, activity.RegisterOptions{Name: names.ActivityNameCreateTempDir})
	w.RegisterActivityWithOptions(acts.RemoveFileActivity, activity.RegisterOptions{Name: names.ActivityNameRemoveFile})

	// Register provider-specific activities
	w.RegisterActivityWithOptions(acts.DownloadActivity, activity.RegisterOptions{Name: names.ActivityNameDownload})
	w.RegisterActivityWithOptions(acts.FTPDownloadActivity, activity.RegisterOptions{Name: names.ActivityNameFileTransferDownload})
	w.RegisterActivityWithOptions(acts.WebDAVDownloadActivity, activity.RegisterOptions{Name: names.ActivityNameWebDAVDownload})
	w.RegisterActivityWithOptions(acts.GitDownloadActivity, activity.RegisterOptions{Name: names.ActivityNameGitDownload})
	w.RegisterActivityWithOptions(acts.MySQLDumpActivity, activity.RegisterOptions{Name: names.ActivityNameMySQLDump})
	w.RegisterActivityWithOptions(acts.PostgreSQLDumpActivity, activity.RegisterOptions{Name: names.ActivityNamePostgreSQLDump})
	w.RegisterActivityWithOptions(acts.MSSQLDumpActivity, activity.RegisterOptions{Name: names.ActivityNameMSSQLDump})
	w.RegisterActivityWithOptions(acts.RedisDumpActivity, activity.RegisterOptions{Name: names.ActivityNameRedisDump})
	w.RegisterActivityWithOptions(acts.AWSS3DownloadActivity, activity.RegisterOptions{Name: names.ActivityNameAWSS3Download})
	w.RegisterActivityWithOptions(acts.AWSDynamoDBDumpActivity, activity.RegisterOptions{Name: names.ActivityNameAWSDynamoDBDump})
	w.RegisterActivityWithOptions(acts.ScriptRunActivity, activity.RegisterOptions{Name: names.ActivityNameScriptRun})

	log.Printf("Loaded %d jobs from config", len(cfg.Jobs))
	for _, job := range cfg.Jobs {
		log.Println("Adding job", job.ID, job.Provider)
	}

	// Start listening to the Task Queue.
	err = w.Run(worker.InterruptCh())
	if err != nil {
		log.Fatalln("unable to start Worker", err)
	}

}
