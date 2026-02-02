package activities

import (
	"context"
	"os"
)

// CleanupActivityInput defines the input for the CleanupActivity
type CleanupActivityInput struct {
	FilePath string
}

// CleanupActivity handles deleting a file
func (a *Activities) CleanupActivity(ctx context.Context, input CleanupActivityInput) error {
	return os.Remove(input.FilePath)
}
