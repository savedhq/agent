package job

import "fmt"

const JobProviderAWSDynamoDB Provider = "aws.dynamodb"

type AWSDynamoDBConfig struct {
	Region          string `json:"region"`
	TableName       string `json:"table_name"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
	BackupMethod    string `json:"backup_method"`
	S3Bucket        string `json:"s3_bucket"`
}

func (c *AWSDynamoDBConfig) Validate() error {
	if c.Region == "" {
		return fmt.Errorf("region is required")
	}
	if c.TableName == "" {
		return fmt.Errorf("table_name is required")
	}
	if c.AccessKeyID == "" {
		return fmt.Errorf("access_key_id is required")
	}
	if c.SecretAccessKey == "" {
		return fmt.Errorf("secret_access_key is required")
	}
	if c.BackupMethod == "export_s3" && c.S3Bucket == "" {
		return fmt.Errorf("s3_bucket is required for export_s3 backup method")
	}
	return nil
}

func (c *AWSDynamoDBConfig) Type() Provider { return JobProviderAWSDynamoDB }
