package activities

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/klauspost/compress/flate"
	"github.com/klauspost/compress/zip"
	"github.com/stretchr/testify/assert"
	"go.temporal.io/sdk/testsuite"
)

func TestFileCompressionActivity(t *testing.T) {
	// Setup test environment
	ts := &testsuite.WorkflowTestSuite{}
	env := ts.NewTestActivityEnvironment()
	a := &Activities{}
	env.RegisterActivity(a)

	// Create a temporary file with some content
	tmpfile, err := os.CreateTemp("", "example")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name()) // clean up

	content := []byte("hello world")
	_, err = tmpfile.Write(content)
	assert.NoError(t, err)
	err = tmpfile.Close()
	assert.NoError(t, err)

	// Execute activity with different compression levels
	for level := flate.NoCompression; level <= flate.BestCompression; level++ {
		future, err := env.ExecuteActivity(a.FileCompressionActivity, FileCompressionActivityInput{
			FilePath:         tmpfile.Name(),
			CompressionLevel: level,
		})
		assert.NoError(t, err)

		var result FileCompressionActivityOutput
		err = future.Get(&result)
		assert.NoError(t, err)


		// Assertions
		expectedPath := tmpfile.Name() + ".zip"
		assert.Equal(t, expectedPath, result.FilePath)

		_, err = os.Stat(result.FilePath)
		assert.NoError(t, err)

		r, err := zip.OpenReader(result.FilePath)
		assert.NoError(t, err)
		defer r.Close()

		assert.Len(t, r.File, 1)
		assert.Equal(t, filepath.Base(tmpfile.Name()), r.File[0].Name)

		f, err := r.File[0].Open()
		assert.NoError(t, err)
		defer f.Close()

		uncompressed, err := io.ReadAll(f)
		assert.NoError(t, err)
		assert.Equal(t, content, uncompressed)
		os.Remove(result.FilePath)
	}
}

func TestFileCompressionActivity_AlreadyCompressed(t *testing.T) {
	// Setup test environment
	ts := &testsuite.WorkflowTestSuite{}
	env := ts.NewTestActivityEnvironment()
	a := &Activities{}
	env.RegisterActivity(a)

	// Execute activity
	future, err := env.ExecuteActivity(a.FileCompressionActivity, FileCompressionActivityInput{
		FilePath:         "test.zip",
		CompressionLevel: 5,
	})
	assert.NoError(t, err)

	var result FileCompressionActivityOutput
	err = future.Get(&result)
	assert.NoError(t, err)

	assert.Equal(t, "test.zip", result.FilePath)
}
