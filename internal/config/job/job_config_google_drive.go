package job

import (
	"errors"
)

const (
	JobProviderGoogleDrive JobProvider = "google.drive"
)

type GoogleDriveConfig struct {
	Email        string   `json:"email"`                   // Google account email
	ClientID     string   `json:"client_id"`               // OAuth2 client ID
	ClientSecret string   `json:"client_secret"`           // OAuth2 client secret
	RefreshToken string   `json:"refresh_token"`           // OAuth2 refresh token
	FolderIDs    []string `json:"folder_ids,omitempty"`    // Specific folder IDs to backup
	FileIDs      []string `json:"file_ids,omitempty"`      // Specific file IDs to backup
	Query        string   `json:"query,omitempty"`         // Drive search query
	BackupAll    bool     `json:"backup_all,omitempty"`    // Backup entire Drive
	SharedDrives bool     `json:"shared_drives,omitempty"` // Include shared drives
}

func (c *GoogleDriveConfig) Validate() error {
	if c.Email == "" {
		return errors.New("email is required")
	}
	if c.ClientID == "" {
		return errors.New("client_id is required")
	}
	if c.ClientSecret == "" {
		return errors.New("client_secret is required")
	}
	if c.RefreshToken == "" {
		return errors.New("refresh_token is required")
	}
	if !c.BackupAll && len(c.FolderIDs) == 0 && len(c.FileIDs) == 0 && c.Query == "" {
		return errors.New("either backup_all must be true or folder_ids/file_ids/query must be specified")
	}
	return nil
}

func (c *GoogleDriveConfig) Type() JobProvider {
	return JobProviderGoogleDrive
}
