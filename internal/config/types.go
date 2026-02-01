package config

import "time"

// BackupResponse represents the response from POST /workspaces/:workspace_id/jobs/:job_id/request
type BackupResponse struct {
	ID          string    `json:"id"`
	WorkspaceId string    `json:"workspace_id"`
	JobId       string    `json:"job_id"`
	Status      string    `json:"status"`
	StartedAt   time.Time `json:"started_at,omitempty"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// FileMeta represents file metadata for backup uploads
type FileMeta struct {
	Size     int64  `json:"size"`
	Checksum string `json:"checksum"`
	Name     string `json:"name"`
	MimeType string `json:"mime_type"`
}

// UploadURLResponse represents the response from POST /workspaces/:workspace_id/jobs/:job_id/upload
type UploadURLResponse struct {
	UploadURL string    `json:"upload_url"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ConfirmResponse represents the response from POST /workspaces/:workspace_id/jobs/:job_id/confirm
type ConfirmResponse struct {
	Status string `json:"status"`
}

// LogConfig represents the logging configuration
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Path       string `mapstructure:"path"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
	Compress   bool   `mapstructure:"compress"`
}
