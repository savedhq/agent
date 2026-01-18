package job

import (
	"encoding/json"
	"fmt"
)

// configFactories maps providers to their config factory functions
var configFactories = map[JobProvider]func() JobConfig{
	JobProviderHTTP:          func() JobConfig { return new(HTTPConfig) },
	JobProviderScript:        func() JobConfig { return new(ScriptConfig) },
	JobProviderAWSS3:         func() JobConfig { return new(AWSS3Config) },
	JobProviderGoogleDrive:   func() JobConfig { return new(GoogleDriveConfig) },
	JobProvideriCloudStorage: func() JobConfig { return new(ICloudStorageConfig) },
}

// NewConfig creates a new empty config for the given provider
func NewConfig(provider JobProvider) (JobConfig, error) {
	factory, ok := configFactories[provider]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}
	return factory(), nil
}

// Marshal converts JobConfig to map[string]any
func Marshal(config JobConfig) (map[string]any, error) {
	if config == nil {
		return nil, nil
	}
	data, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Unmarshal deserializes bytes into the appropriate JobConfig type
func Unmarshal(provider JobProvider, data []byte) (JobConfig, error) {
	config, err := NewConfig(provider)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return config, nil
}

// FromMap loads JobConfig from map[string]any
func FromMap(provider JobProvider, m map[string]any) (JobConfig, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return Unmarshal(provider, data)
}

// LoadAs extracts and type-asserts the config from a Job
func LoadAs[T JobConfig](job Job) (T, error) {
	var zero T
	config := job.Config
	if config == nil {
		return zero, fmt.Errorf("config is nil")
	}
	result, ok := config.(T)
	if !ok {
		return zero, fmt.Errorf("failed to cast config to %T (actual type: %T)", zero, config)
	}
	return result, nil
}
