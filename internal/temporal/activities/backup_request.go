package activities

import (
	"agent/internal/job"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
)

type BackupRequestActivityInput struct {
	Job *job.Job `json:"job"`
}

type BackupRequestActivityOutput struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceId uuid.UUID `json:"workspace_id"`
	JobId       uuid.UUID `json:"job_id"`
}

func (a *Activities) BackupRequestActivity(ctx context.Context, input BackupRequestActivityInput) (*BackupRequestActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("BackupRequestActivity called", "jobId", input.Job.ID)

	// Get auth token
	token, err := a.Auth.Token()
	if err != nil {
		logger.Error("Failed to get auth token", "error", err)
		return nil, fmt.Errorf("failed to get auth token: %w", err)
	}

	// Build API URL
	url := fmt.Sprintf("%s/v1/workspaces/%s/jobs/%s/request",
		a.Config.API,
		a.Hub.Workspace,
		input.Job.ID)

	info := activity.GetInfo(ctx)
	runID := info.WorkflowExecution.ID

	logger.Info("Sending backup request", "url", url, "runId", runID)

	// Create request body
	reqBody := map[string]string{
		"run_id": runID,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		logger.Error("HTTP request failed", "url", url, "error", err)
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	logger.Info("Backup request response", "status", resp.StatusCode, "body", string(body))

	// Check status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result BackupRequestActivityOutput
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	logger.Info("Backup request created", "backupId", result.ID.String())
	return &result, nil
}
