package activities

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/flate"
	"github.com/klauspost/compress/zip"
	"go.temporal.io/sdk/activity"
)

type FileCompressionActivityInput struct {
	FilePath         string
	CompressionLevel int
}

type FileCompressionActivityOutput struct {
	FilePath string
}

func (a *Activities) FileCompressionActivity(ctx context.Context, input FileCompressionActivityInput) (*FileCompressionActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Debug("FileCompressionActivity called")

	// Skip compression for already-compressed formats
	if isAlreadyCompressed(input.FilePath) {
		logger.Info("File is already compressed, skipping compression", "filePath", input.FilePath)
		return &FileCompressionActivityOutput{
			FilePath: input.FilePath,
		}, nil
	}
	// 1. Create a new zip file
	newZipFile, err := os.Create(input.FilePath + ".zip")
	if err != nil {
		return nil, fmt.Errorf("failed to create zip file: %w", err)
	}
	defer newZipFile.Close()

	// 2. Create a new zip writer
	zipWriter := zip.NewWriter(newZipFile)
	defer zipWriter.Close()

	// 3. Set compression level
	level := input.CompressionLevel
	if level == 0 {
		level = flate.DefaultCompression
	}
	zipWriter.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(out, level)
	})

	// 4. Add the file to the zip
	if err := addFileToZip(zipWriter, input.FilePath); err != nil {
		return nil, fmt.Errorf("failed to add file to zip: %w", err)
	}

	result := &FileCompressionActivityOutput{
		FilePath: input.FilePath + ".zip",
	}

	return result, nil
}

func addFileToZip(zipWriter *zip.Writer, filename string) error {
	// 1. Open the file to be added
	fileToZip, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer fileToZip.Close()

	// 2. Get the file info
	info, err := fileToZip.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// 3. Create a new zip file header
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return fmt.Errorf("failed to create zip header: %w", err)
	}

	// 4. Set the compression method
	header.Method = zip.Deflate
	header.Name = filepath.Base(filename)

	// 5. Create a new zip writer for the file
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("failed to create zip writer: %w", err)
	}

	// 6. Copy the file contents to the zip writer
	_, err = io.Copy(writer, fileToZip)
	if err != nil {
		return fmt.Errorf("failed to copy file to zip: %w", err)
	}

	return nil
}

func isAlreadyCompressed(filename string) bool {
	compressedExtensions := []string{".zip", ".gz", ".bz2", ".rar", ".7z"}
	for _, ext := range compressedExtensions {
		if strings.HasSuffix(filename, ext) {
			return true
		}
	}
	return false
}
