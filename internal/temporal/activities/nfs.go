package activities

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

type MountActivityInput struct {
	RemotePath string
	LocalPath  string
}

type MountActivityOutput struct{}

func (a *Activities) MountActivity(ctx context.Context, input MountActivityInput) (*MountActivityOutput, error) {
	// Create the local directory if it doesn't exist.
	if err := os.MkdirAll(input.LocalPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create mount point directory %s: %w", input.LocalPath, err)
	}

	// Execute the mount command.
	cmd := exec.Command("mount", "-t", "nfs", input.RemotePath, input.LocalPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("failed to mount %s to %s: %w, output: %s", input.RemotePath, input.LocalPath, err, string(output))
	}

	return &MountActivityOutput{}, nil
}

type UnmountActivityInput struct {
	LocalPath string
}

type UnmountActivityOutput struct{}

func (a *Activities) UnmountActivity(ctx context.Context, input UnmountActivityInput) (*UnmountActivityOutput, error) {
	cmd := exec.Command("umount", input.LocalPath)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to unmount %s: %w", input.LocalPath, err)
	}
	return &UnmountActivityOutput{}, nil
}
