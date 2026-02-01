package activities

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.temporal.io/sdk/testsuite"
)

func TestFileCleanupActivity(t *testing.T) {
	// Setup test environment
	ts := &testsuite.WorkflowTestSuite{}
	env := ts.NewTestActivityEnvironment()
	a := &Activities{}
	env.RegisterActivity(a)

	// Create a temporary file
	tmpfile, err := os.CreateTemp("", "example")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name()) // clean up in case of test failure

	// Execute activity
	_, err = env.ExecuteActivity(a.FileCleanupActivity, FileCleanupActivityInput{
		FilePath: tmpfile.Name(),
	})
	assert.NoError(t, err)

	// Assertions
	_, err = os.Stat(tmpfile.Name())
	assert.True(t, os.IsNotExist(err))
}
