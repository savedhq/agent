
package job

import (
	"errors"
	"fmt"
	"time"
)

// ScriptProvider is the provider name for script backups
const ScriptProvider JobProvider = "script"

// Script represents the configuration for a script-based backup job
type Script struct {
	Path    string        `mapstructure:"path"`
	Timeout time.Duration `mapstructure:"timeout"`
}

// Type returns the provider type for the Script config
func (s *Script) Type() JobProvider {
	return ScriptProvider
}

// Validate checks if the Script configuration is valid
func (s *Script) Validate() error {
	if s.Path == "" {
		return errors.New("script path is required")
	}
	if s.Timeout <= 0 {
		return fmt.Errorf("invalid timeout value: %s", s.Timeout)
	}
	return nil
}
