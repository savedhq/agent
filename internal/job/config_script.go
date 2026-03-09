package job

import "errors"

const JobProviderScript Provider = "script"

type ScriptConfig struct {
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
	WorkDir string   `json:"workdir,omitempty"`
}

func (c *ScriptConfig) Validate() error {
	if c.Command == "" {
		return errors.New("command is required")
	}
	return nil
}

func (c *ScriptConfig) Type() Provider { return JobProviderScript }
