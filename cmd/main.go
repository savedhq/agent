package main

import (
	"agent/internal/auth"
	"agent/internal/config"
	"agent/internal/temporal/activities"
	"agent/internal/temporal/workflows"
	"agent/pkg"
	"agent/pkg/names"
	"agent/pkg/log"
	"context"
	"crypto/tls"
	"os"

	"github.com/rs/zerolog"
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
		// Can't use logger here because it's not initialized yet
		panic(err)
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logger := log.New(cfg.Log)

	authService := auth.NewAuthService(&cfg.Auth)

	// Fetch hub configuration from API
	hubConfig, err := pkg.LoadHubConfig(ctx, authService, cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to load hub config")
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
		logger.Fatal().Err(err).Msg("Unable to create Temporal client")
	}
	logger.Info().Str("workspace", hubConfig.Workspace).Str("queue", hubConfig.Queue).Msg("Connected to workspace")

	defer c.Close()

	workerOptions := worker.Options{
		EnableSessionWorker: true,
		Logger:              log.NewTemporalAdapter(logger),
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

	logger.Info().Int("count", len(cfg.Jobs)).Msg("Loaded jobs from config")
	for _, job := range cfg.Jobs {
		logger.Info().Str("jobId", job.ID.String()).Str("provider", string(job.Provider)).Msg("Adding job")
	}

	// Start listening to the Task Queue.
	err = w.Run(worker.InterruptCh())
	if err != nil {
		logger.Fatal().Err(err).Msg("Unable to start Worker")
	}

}
