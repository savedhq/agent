package activities

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"

	"go.temporal.io/sdk/activity"
)

type FileCompressionActivityInput struct {
	FilePath string
}

type FileCompressionActivityOutput struct {
	FilePath string
	Name     string
	Size     int64
	Checksum string
}

func (a *Activities) FileCompressionActivity(ctx context.Context, input FileCompressionActivityInput) (*FileCompressionActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("FileCompressionActivity started", "path", input.FilePath)

	// Create a new zip file
	zipFile, err := os.CreateTemp("", "backup-*.zip")
	if err != nil {
		return nil, err
	}

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Walk through the directory and add files to the zip
	err = filepath.Walk(input.FilePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Create a proper relative path for the file in the zip archive
		relPath, err := filepath.Rel(input.FilePath, path)
		if err != nil {
			return err
		}
		// Use forward slashes for zip archive paths for compatibility
		relPath = filepath.ToSlash(relPath)

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = relPath
		header.Method = zip.Deflate // Use compression

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})

	if err != nil {
		return nil, err
	}

	// Must close the zipWriter to flush all data to the underlying writer (the file)
	if err := zipWriter.Close(); err != nil {
		return nil, err
	}
    // Must close the file before calculating stats
    if err := zipFile.Close(); err != nil {
        return nil, err
    }


	// Get file info for size
	fileInfo, err := os.Stat(zipFile.Name())
	if err != nil {
		return nil, err
	}

	// Calculate checksum
	file, err := os.Open(zipFile.Name())
	if err != nil {
		return nil, err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, err
	}
	checksum := hex.EncodeToString(hash.Sum(nil))

	output := &FileCompressionActivityOutput{
		FilePath: zipFile.Name(),
		Name:     filepath.Base(zipFile.Name()),
		Size:     fileInfo.Size(),
		Checksum: checksum,
	}

	logger.Info("FileCompressionActivity completed", "output", output)

	return output, nil
}
