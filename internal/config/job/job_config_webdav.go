package job

import "errors"

// WebDAVJobConfig holds the configuration for a WebDAV backup job
type WebDAVJobConfig struct {
	URL                string `json:"url"`
	Username           string `json:"username"`
	Password           string `json:"password"`
	InsecureSkipVerify bool   `json:"insecure_skip_verify"`
}

// Type returns the job provider type for WebDAV
func (c *WebDAVJobConfig) Type() JobProvider {
	return JobProviderWebDAV
}

// Validate checks if the WebDAVJobConfig is valid
func (c *WebDAVJobConfig) Validate() error {
	if c.URL == "" {
		return errors.New("WebDAV URL is required")
	}
	return nil
}
