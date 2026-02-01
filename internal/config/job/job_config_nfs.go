package job

import "errors"

const (
	JobProviderNFS JobProvider = "nfs"
)

type NFSConfig struct {
	RemotePath string `json:"remote_path"`
	LocalPath  string `json:"local_path"`
}

func (c *NFSConfig) Validate() error {
	if c.RemotePath == "" {
		return errors.New("remote_path is required")
	}
	if c.LocalPath == "" {
		return errors.New("local_path is required")
	}
	return nil
}

func (c *NFSConfig) Type() JobProvider {
	return JobProviderNFS
}
