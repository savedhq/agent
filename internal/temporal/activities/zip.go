package activities

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ZipActivityInput defines the input for the ZipActivity.
type ZipActivityInput struct {
	SourcePath      string `json:"source_path"`
	DestinationPath string `json:"destination_path"`
}

// ZipActivityOutput defines the output for the ZipActivity.
type ZipActivityOutput struct {
	FilePath string `json:"file_path"`
	Size     int64  `json:"size"`
	Checksum string `json:"checksum"`
	Name     string `json:"name"`
	MimeType string `json:"mime_type"`
}

// ZipActivity compresses a directory into a zip file.
func (a *Activities) ZipActivity(ctx context.Context, input ZipActivityInput) (*ZipActivityOutput, error) {
	zipFile, err := os.Create(input.DestinationPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create zip file %s: %w", input.DestinationPath, err)
	}

	zipWriter := zip.NewWriter(zipFile)

	err = filepath.WalkDir(input.SourcePath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil // Skip directories
		}

		relPath, err := filepath.Rel(input.SourcePath, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", path, err)
		}

		zipEntry, err := zipWriter.Create(relPath)
		if err != nil {
			return fmt.Errorf("failed to create zip entry for %s: %w", relPath, err)
		}

		fileToZip, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", path, err)
		}
		defer fileToZip.Close()

		_, err = io.Copy(zipEntry, fileToZip)
		if err != nil {
			return fmt.Errorf("failed to copy file content for %s: %w", path, err)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed during zip creation for %s: %w", input.SourcePath, err)
	}

	// Close the writer to ensure all data is flushed to the file before we read it.
	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close zip writer: %w", err)
	}
    if err := zipFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close zip file handle: %w", err)
	}

	// Get file info for size and calculate checksum.
	fileInfo, err := os.Stat(input.DestinationPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info for %s: %w", input.DestinationPath, err)
	}

	file, err := os.Open(input.DestinationPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open zip file for checksum calculation %s: %w", input.DestinationPath, err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, fmt.Errorf("failed to calculate checksum for %s: %w", input.DestinationPath, err)
	}
	checksum := fmt.Sprintf("%x", hash.Sum(nil))

	output := &ZipActivityOutput{
		FilePath: input.DestinationPath,
		Size:     fileInfo.Size(),
		Checksum: checksum,
		Name:     filepath.Base(input.DestinationPath),
		MimeType: "application/zip",
	}

	return output, nil
}
