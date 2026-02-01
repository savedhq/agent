package activities

import (
	"agent/internal/config/job"
	"context"
	"fmt"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

type GitCloneActivityInput struct {
	Job *job.Job
}

type GitCloneActivityOutput struct {
	Path string
}

func (a *Activities) GitCloneActivity(ctx context.Context, input GitCloneActivityInput) (*GitCloneActivityOutput, error) {
	gitConfig, ok := input.Job.Config.(*job.GitConfig)
	if !ok {
		return nil, fmt.Errorf("invalid git config")
	}

	// Create a temporary directory for the clone
	tempDir, err := os.MkdirTemp("", "git-clone-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	cloneOptions := &git.CloneOptions{
		URL: gitConfig.URL,
	}

	if gitConfig.Auth != nil {
		if gitConfig.Auth.SSHPrivateKey != "" {
			publicKey, err := ssh.NewPublicKeys("git", []byte(gitConfig.Auth.SSHPrivateKey), "")
			if err != nil {
				return nil, fmt.Errorf("failed to create public keys: %w", err)
			}
			cloneOptions.Auth = publicKey
		} else if gitConfig.Auth.HTTPSUsername != "" {
			cloneOptions.Auth = &http.BasicAuth{
				Username: gitConfig.Auth.HTTPSUsername,
				Password: gitConfig.Auth.HTTPSPassword,
			}
		}
	}

	if gitConfig.Shallow {
		cloneOptions.Depth = 1
	}

	if gitConfig.Submodules {
		cloneOptions.RecurseSubmodules = git.DefaultSubmoduleRecursionDepth
	}

	_, err = git.PlainClone(tempDir, false, cloneOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	return &GitCloneActivityOutput{Path: tempDir}, nil
}
