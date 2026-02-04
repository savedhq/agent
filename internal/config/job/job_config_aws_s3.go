package job

import "errors"

const (
	JobProviderAWSS3 JobProvider = "aws.s3"
)

// AWSS3Config defines the configuration for an S3 backup job.
type AWSS3Config struct {
	Region          string `json:"region"`
	Bucket          string `json:"bucket"`
	Path            string `json:"path"`
	AccessKeyID     string `json:"access_key_id,omitempty"`
	SecretAccessKey string `json:"secret_access_key,omitempty"`
	Endpoint        string `json:"endpoint,omitempty"`
}

func (c *AWSS3Config) Validate() error {
	if c.Region == "" && c.Endpoint == "" {
		return errors.New("region or endpoint is required")
	}
	if c.Bucket == "" {
		return errors.New("bucket is required")
	}
	return nil
}

func (c *AWSS3Config) Type() JobProvider {
	return JobProviderAWSS3
}
