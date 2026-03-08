package activities

import (
	"agent/internal/job"
	"archive/zip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func (a *Activities) s3DownloadAsZip(ctx context.Context, client *s3.Client, cfg *job.AWSS3Config, jobID string, objects []s3types.Object) (*DownloadActivityOutput, error) {
	fileName := fmt.Sprintf("%s.zip", jobID)
	tempFilePath := filepath.Join(a.Config.TempDir, fileName)

	zipFile, err := os.Create(tempFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, obj := range objects {
		if strings.HasSuffix(*obj.Key, "/") {
			continue
		}
		resp, err := client.GetObject(ctx, &s3.GetObjectInput{
			Bucket: aws.String(cfg.Bucket),
			Key:    obj.Key,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to download object %s: %w", *obj.Key, err)
		}

		relPath, err := filepath.Rel(cfg.Path, *obj.Key)
		if err != nil || strings.Contains(relPath, "..") {
			relPath = filepath.Base(*obj.Key)
		}

		f, err := zipWriter.Create(relPath)
		if err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to create zip entry for %s: %w", *obj.Key, err)
		}
		if _, err := io.Copy(f, resp.Body); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to write zip entry %s: %w", *obj.Key, err)
		}
		resp.Body.Close()
	}

	zipWriter.Close()
	zipFile.Close()

	return a.hashAndReturn(tempFilePath, fileName, "application/zip")
}

// hashAndReturn computes SHA256 hash and returns DownloadActivityOutput
func (a *Activities) hashAndReturn(filePath, name, mimeType string) (*DownloadActivityOutput, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for hashing: %w", err)
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, fmt.Errorf("failed to calculate hash: %w", err)
	}

	return &DownloadActivityOutput{
		FilePath: filePath,
		Size:     fi.Size(),
		Checksum: fmt.Sprintf("%x", hash.Sum(nil)),
		Name:     name,
		MimeType: mimeType,
	}, nil
}
