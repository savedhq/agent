package activities

import (
	"agent/internal/config/job"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

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
	logger.Debug("BackupRequestActivity called", "jobId", input.Job.ID)

	// Get auth token
	token, err := a.Auth.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get auth token: %w", err)
	}

	// Build API URL
	url := fmt.Sprintf("%s/v1/workspaces/%s/jobs/%s/request",
		a.Config.API,
		a.Hub.Workspace,
		input.Job.ID)

	// Get Temporal Run ID
	info := activity.GetInfo(ctx)
	runID := info.WorkflowExecution.RunID

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

type BackupUploadActivityInput struct {
	JobId    string `json:"job_id"`
	BackupId string `json:"backup_id"`
	Size     int64  `json:"size"`
	Checksum string `json:"checksum"`
	Name     string `json:"name"`
	MimeType string `json:"mime_type"`
	FilePath string `json:"file_path"`
}

type BackupUploadActivityOutput struct {
	UploadURL string    `json:"upload_url"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (a *Activities) BackupUploadActivity(ctx context.Context, input BackupUploadActivityInput) (*BackupUploadActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Debug("BackupUploadActivity called", "jobId", input.JobId, "backupId", input.BackupId)

	// Get auth token
	token, err := a.Auth.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get auth token: %w", err)
	}

	// Build API URL
	url := fmt.Sprintf("%s/v1/workspaces/%s/jobs/%s/backups/%s/upload",
		a.Config.API,
		a.Hub.Workspace,
		input.JobId,
		input.BackupId)

	// Create file metadata request body
	fileMeta := map[string]interface{}{
		"size":      input.Size,
		"checksum":  input.Checksum,
		"name":      input.Name,
		"mime_type": input.MimeType,
	}

	bodyBytes, err := json.Marshal(fileMeta)
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

	// Parse response
	var result BackupUploadActivityOutput
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	logger.Info("Upload URL received", "expiresAt", result.ExpiresAt)
	return &result, nil
}

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
