package activities

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.temporal.io/sdk/testsuite"
)

func TestGetFileMetadataActivity(t *testing.T) {
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

	// Execute activity
	future, err := env.ExecuteActivity(a.GetFileMetadataActivity, GetFileMetadataActivityInput{
		FilePath: tmpfile.Name(),
	})
	assert.NoError(t, err)

	var result GetFileMetadataActivityOutput
	err = future.Get(&result)
	assert.NoError(t, err)

	// Assertions
	hasher := sha256.New()
	hasher.Write(content)
	expectedChecksum := hex.EncodeToString(hasher.Sum(nil))

	stat, err := os.Stat(tmpfile.Name())
	assert.NoError(t, err)

	assert.Equal(t, stat.Size(), result.Size)
	assert.Equal(t, expectedChecksum, result.Checksum)
}
