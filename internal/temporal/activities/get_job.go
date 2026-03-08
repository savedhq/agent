package activities

import (
	"agent/internal/job"
	"context"
	"fmt"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
)

type GetJobActivityInput struct {
	JobId string `json:"job_id"`
}

type GetJobActivityOutput struct {
	Job *job.Job
}

func (a *Activities) GetJobActivity(ctx context.Context, input GetJobActivityInput) (*GetJobActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("GetJobActivity called", "jobId", input.JobId)

	result := new(GetJobActivityOutput)

	// Find job in config by ID
	for i := range a.Config.Jobs {
		logger.Debug("Checking job", "configJobId", a.Config.Jobs[i].ID, "inputJobId", input.JobId)
		if a.Config.Jobs[i].ID == input.JobId {
			result.Job = &a.Config.Jobs[i]
			logger.Info("Job found", "jobId", input.JobId, "provider", result.Job.Provider)
			return result, nil
		}
	}

	return nil, temporal.NewNonRetryableApplicationError(
		fmt.Sprintf("job not found: %s", input.JobId), "JobNotFound", nil)
}
