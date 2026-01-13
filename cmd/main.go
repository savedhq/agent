package main

import (
	"agent/internal/auth"
	"agent/internal/config"
	"agent/internal/temporal/activities"
	"agent/internal/temporal/workflows"
	"agent/pkg"
	"agent/pkg/names"
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

	authService := auth.NewAuthService(&cfg.Auth)

	// Fetch hub configuration from API
	hubConfig, err := pkg.LoadHubConfig(ctx, authService, cfg)
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

	// Register workflows (same names as backend)
	w.RegisterWorkflowWithOptions(workflows.HTTPBackupWorkflow, workflow.RegisterOptions{Name: names.WorkflowNameHTTP})

	// Create activities instance with dependency injection
	acts := activities.NewActivities(cfg, authService, *hubConfig)

	// Register activities
	w.RegisterActivityWithOptions(acts.BackupRequestActivity, activity.RegisterOptions{Name: names.ActivityNameBackupRequest})
	w.RegisterActivityWithOptions(acts.BackupUploadActivity, activity.RegisterOptions{Name: names.ActivityNameBackupUpload})
	w.RegisterActivityWithOptions(acts.BackupConfirmActivity, activity.RegisterOptions{Name: names.ActivityNameBackupConfirm})
	w.RegisterActivityWithOptions(acts.FileCompressionActivity, activity.RegisterOptions{Name: names.ActivityNameCompressFile})
	w.RegisterActivityWithOptions(acts.DownloadActivity, activity.RegisterOptions{Name: names.ActivityNameDownload})
	w.RegisterActivityWithOptions(acts.FileEncryptionActivity, activity.RegisterOptions{Name: names.ActivityNameEncryptFile})
	w.RegisterActivityWithOptions(acts.GetJobActivity, activity.RegisterOptions{Name: names.ActivityNameGetJob})
	w.RegisterActivityWithOptions(acts.FileUploadS3Activity, activity.RegisterOptions{Name: names.ActivityNameFileUploadS3})

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
