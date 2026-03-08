package activities

import (
	"agent/internal/job"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.temporal.io/sdk/testsuite"
)

func TestScriptRunActivity(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestActivityEnvironment()

	// Mock Activities struct
	// We don't need actual services for this test as ScriptRunActivity doesn't use them
	acts := &Activities{}
	env.RegisterActivity(acts.ScriptRunActivity)

	// Create a dummy script that prints to stdout
	tmpScript, err := os.CreateTemp("", "test-script-*.sh")
	assert.NoError(t, err)
	defer os.Remove(tmpScript.Name())

	content := []byte("#!/bin/sh\necho -n 'hello world'")
	_, err = tmpScript.Write(content)
	assert.NoError(t, err)
	tmpScript.Close()
	os.Chmod(tmpScript.Name(), 0755)

	jobConfig := &job.Job{
		ID: "test-job-1",
		Script: &job.ScriptConfig{
			Command: tmpScript.Name(),
		},
	}

	input := ScriptRunActivityInput{
		Job: jobConfig,
	}

	val, err := env.ExecuteActivity(acts.ScriptRunActivity, input)
	assert.NoError(t, err)

	var res ScriptRunActivityOutput
	err = val.Get(&res)
	assert.NoError(t, err)

	assert.FileExists(t, res.FilePath)
	defer os.Remove(res.FilePath)

	// Verify content
	outContent, err := os.ReadFile(res.FilePath)
	assert.NoError(t, err)
	assert.Equal(t, "hello world", string(outContent))

	// Verify metadata
	assert.Equal(t, int64(11), res.Size)
	assert.Equal(t, "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9", res.Checksum) // SHA256 of "hello world"
	assert.Equal(t, "application/octet-stream", res.MimeType)
}
