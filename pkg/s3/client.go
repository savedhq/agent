package s3

import (
  "context"
  "fmt"

  "github.com/aws/aws-sdk-go-v2/aws"
  "github.com/aws/aws-sdk-go-v2/config"
  "github.com/aws/aws-sdk-go-v2/credentials"
  "github.com/aws/aws-sdk-go-v2/service/s3"
)

// NewClient creates a new S3 client with the provided configuration
func NewClient(region, endpoint, accessKeyID, secretAccessKey string) (*s3.Client, error) {
  // 1. Load the default config without the endpoint resolver
  cfg, err := config.LoadDefaultConfig(
    context.Background(),
    config.WithRegion(region),
    config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
  )
  if err != nil {
    return nil, fmt.Errorf("failed to load AWS config: %w", err)
  }

  // 2. Pass the endpoint configuration to the client constructor
  return s3.NewFromConfig(cfg, func(o *s3.Options) {
    if endpoint != "" {
      o.BaseEndpoint = aws.String(endpoint)
    }

    // Recommended for 3rd party S3 providers (MinIO, etc.) to force path-style addressing
    o.UsePathStyle = true
  }), nil
}
