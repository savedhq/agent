package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.temporal.io/sdk/activity"
)

type BackupConfirmActivityInput struct {
	JobId    string `json:"job_id"`
	BackupId string `json:"backup_id"`
	Status   bool   `json:"status"`
}

type BackupConfirmActivityOutput struct {
	Status bool `json:"status"`
}

func (a *Activities) BackupConfirmActivity(ctx context.Context, input BackupConfirmActivityInput) (*BackupConfirmActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Debug("BackupConfirmActivity called", "jobId", input.JobId, "backupId", input.BackupId)

	// Get auth token
	token, err := a.Auth.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get auth token: %w", err)
	}

	// Build API URL
	url := fmt.Sprintf("%s/v1/workspaces/%s/jobs/%s/backups/%s/confirm",
		a.Config.API,
		a.Hub.Workspace,
		input.JobId,
		input.BackupId)

	// Create request body
	reqBody := map[string]interface{}{
		"status": input.Status,
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
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	result := &BackupConfirmActivityOutput{
		Status: input.Status,
	}

	logger.Info("Backup confirmed", "status", result.Status)
	return result, nil
}
