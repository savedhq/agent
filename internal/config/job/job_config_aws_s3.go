package job

import (
	"errors"
	"fmt"
)

const (
	JobProviderAWSS3 JobProvider = "aws.s3"
)

// S3LocationConfig defines the configuration for a single S3 location (source or destination).
type S3LocationConfig struct {
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
	Path            string `json:"path"` // Optional prefix
	AccessKeyID     string `json:"access_key_id,omitempty"`
	SecretAccessKey string `json:"secret_access_key,omitempty"`
	Endpoint        string `json:"endpoint,omitempty"` // For S3-compatible services
}

// AWSS3Config defines the configuration for an S3 backup job.
type AWSS3Config struct {
	Source      S3LocationConfig `json:"source"`
	Destination S3LocationConfig `json:"destination"`
}

func (c *S3LocationConfig) Validate() error {
	if c.Region == "" && c.Endpoint == "" {
		return errors.New("region or endpoint is required")
	}
	if c.Bucket == "" {
		return errors.New("bucket is required")
	}
	return nil
}

func (c *AWSS3Config) Validate() error {
	if err := c.Source.Validate(); err != nil {
		return fmt.Errorf("source config validation failed: %w", err)
	}
	if err := c.Destination.Validate(); err != nil {
		return fmt.Errorf("destination config validation failed: %w", err)
	}
	return nil
}

func (c *AWSS3Config) Type() JobProvider {
	return JobProviderAWSS3
}
