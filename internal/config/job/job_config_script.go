package job

import (
	"errors"
)

const (
	JobProviderScript JobProvider = "script"
)

type ScriptConfig struct {
	Command string   `json:"command"`           // Command to execute
	Args    []string `json:"args,omitempty"`    // Command arguments
	WorkDir string   `json:"workdir,omitempty"` // Working directory
}

func (c *ScriptConfig) Validate() error {
	if c.Command == "" {
		return errors.New("command is required")
	}
	return nil
}

func (c *ScriptConfig) Type() JobProvider {
	return JobProviderScript
}
