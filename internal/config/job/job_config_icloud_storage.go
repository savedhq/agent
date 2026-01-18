package job

import (
	"errors"
)

const (
	JobProvideriCloudStorage JobProvider = "icloud.storage"
)

type ICloudStorageConfig struct {
	AppleID       string   `json:"apple_id"`                 // Apple ID email
	Password      string   `json:"password"`                 // App-specific password
	Folders       []string `json:"folders,omitempty"`        // Specific folders to backup
	BackupAll     bool     `json:"backup_all,omitempty"`     // Backup all iCloud Drive
	IncludeShared bool     `json:"include_shared,omitempty"` // Include shared folders
}

func (c *ICloudStorageConfig) Validate() error {
	if c.AppleID == "" {
		return errors.New("apple_id is required")
	}
	if c.Password == "" {
		return errors.New("password is required")
	}
	if !c.BackupAll && len(c.Folders) == 0 {
		return errors.New("either backup_all must be true or folders must be specified")
	}
	return nil
}

func (c *ICloudStorageConfig) Type() JobProvider {
	return JobProvideriCloudStorage
}
