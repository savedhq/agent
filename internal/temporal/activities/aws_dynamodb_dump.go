package activities

import (
	"agent/internal/job"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"go.temporal.io/sdk/activity"
)

type AWSDynamoDBDumpActivityInput struct {
	Job *job.Job `json:"job"`
}

func (a *Activities) AWSDynamoDBDumpActivity(ctx context.Context, input AWSDynamoDBDumpActivityInput) (*DownloadActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("AWSDynamoDBDumpActivity started", "jobId", input.Job.ID)

	dynamoConfig, err := job.LoadAs[*job.AWSDynamoDBConfig](*input.Job)
	if err != nil {
		return nil, fmt.Errorf("failed to load DynamoDB config: %w", err)
	}
	if err := dynamoConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid DynamoDB config: %w", err)
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(dynamoConfig.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(dynamoConfig.AccessKeyID, dynamoConfig.SecretAccessKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load aws config: %w", err)
	}

	client := dynamodb.NewFromConfig(cfg)

	fileName := fmt.Sprintf("%s.json", input.Job.ID)
	filePath := filepath.Join(a.Config.TempDir, fileName)

	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if _, err := file.WriteString("["); err != nil {
		return nil, fmt.Errorf("failed to write start of json array: %w", err)
	}

	paginator := dynamodb.NewScanPaginator(client, &dynamodb.ScanInput{
		TableName: aws.String(dynamoConfig.TableName),
	})

	firstItem := true
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to scan dynamodb table: %w", err)
		}
		for _, item := range page.Items {
			if !firstItem {
				if _, err := file.WriteString(","); err != nil {
					return nil, fmt.Errorf("failed to write comma: %w", err)
				}
			}
			simpleMap := make(map[string]any)
			for k, v := range item {
				simpleMap[k] = unmarshalAV(v)
			}
			if err := encoder.Encode(simpleMap); err != nil {
				return nil, fmt.Errorf("failed to encode item: %w", err)
			}
			firstItem = false
		}
	}

	if _, err := file.WriteString("]"); err != nil {
		return nil, fmt.Errorf("failed to write end of json array: %w", err)
	}

	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek file: %w", err)
	}

	fi, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, fmt.Errorf("failed to calculate hash: %w", err)
	}

	logger.Info("AWSDynamoDBDumpActivity completed", "filePath", filePath, "size", fi.Size())

	return &DownloadActivityOutput{
		FilePath: filePath,
		Size:     fi.Size(),
		Checksum: fmt.Sprintf("%x", hash.Sum(nil)),
		Name:     fileName,
		MimeType: "application/json",
	}, nil
}

func unmarshalAV(av types.AttributeValue) any {
	switch v := av.(type) {
	case *types.AttributeValueMemberS:
		return v.Value
	case *types.AttributeValueMemberN:
		return v.Value
	case *types.AttributeValueMemberBOOL:
		return v.Value
	case *types.AttributeValueMemberNULL:
		return nil
	case *types.AttributeValueMemberM:
		result := make(map[string]any)
		for k, val := range v.Value {
			result[k] = unmarshalAV(val)
		}
		return result
	case *types.AttributeValueMemberL:
		result := make([]any, len(v.Value))
		for i, val := range v.Value {
			result[i] = unmarshalAV(val)
		}
		return result
	case *types.AttributeValueMemberBS:
		return v.Value
	case *types.AttributeValueMemberNS:
		return v.Value
	case *types.AttributeValueMemberSS:
		return v.Value
	case *types.AttributeValueMemberB:
		return v.Value
	default:
		return nil
	}
}
