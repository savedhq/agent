package job

import (
	"encoding/json"
	"fmt"

	"github.com/go-viper/mapstructure/v2"
)

var configFactories = map[Provider]func() Config{
	JobProviderHTTP:        func() Config { return new(HTTPConfig) },
	JobProviderFTP:         func() Config { return new(FTPConfig) },
	JobProviderSFTP:        func() Config { return new(SFTPConfig) },
	JobProviderWebDAV:      func() Config { return new(WebDAVConfig) },
	JobProviderGit:         func() Config { return new(GitConfig) },
	JobProviderAWSS3:       func() Config { return new(AWSS3Config) },
	JobProviderAWSDynamoDB: func() Config { return new(AWSDynamoDBConfig) },
	JobProviderMySQL:       func() Config { return new(MySQLConfig) },
	JobProviderPostgreSQL:  func() Config { return new(PostgreSQLConfig) },
	JobProviderMSSQL:       func() Config { return new(MSSQLConfig) },
	JobProviderRedis:       func() Config { return new(RedisConfig) },
	JobProviderIMAP:        func() Config { return new(IMAPConfig) },
	JobProviderScript:      func() Config { return new(ScriptConfig) },
}

// LoadAs safely casts the job config to T
func LoadAs[T Config](j Job) (T, error) {
	config, ok := j.Config.(T)
	if !ok {
		var zero T
		return zero, fmt.Errorf("job config is not of expected type")
	}
	return config, nil
}

// jobJSON is a helper for serializing/deserializing Job with a typed config.
type jobJSON struct {
	ID          string            `json:"id"`
	Provider    Provider          `json:"provider"`
	Config      json.RawMessage   `json:"config"`
	Encryption  EncryptionConfig  `json:"encryption"`
	Compression CompressionConfig `json:"compression"`
	Script      *ScriptConfig     `json:"script,omitempty"`
}

func (j *Job) MarshalJSON() ([]byte, error) {
	raw, err := json.Marshal(j.Config)
	if err != nil {
		return nil, err
	}
	return json.Marshal(jobJSON{
		ID: j.ID, Provider: j.Provider, Config: raw,
		Encryption: j.Encryption, Compression: j.Compression, Script: j.Script,
	})
}

func (j *Job) UnmarshalJSON(data []byte) error {
	var tmp jobJSON
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	factory, ok := configFactories[tmp.Provider]
	if !ok {
		return fmt.Errorf("unknown provider: %s", tmp.Provider)
	}
	cfg := factory()
	if err := json.Unmarshal(tmp.Config, cfg); err != nil {
		return err
	}
	j.ID = tmp.ID
	j.Provider = tmp.Provider
	j.Config = cfg
	j.Encryption = tmp.Encryption
	j.Compression = tmp.Compression
	j.Script = tmp.Script
	return nil
}

// ConfigFromMap loads a typed Config from a raw map using the provider's factory.
// Uses mapstructure with WeaklyTypedInput so YAML string values (e.g. port: "22") are
// coerced to the correct Go types instead of failing JSON unmarshal.
func ConfigFromMap(provider Provider, m map[string]any) (Config, error) {
	factory, ok := configFactories[provider]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}

	cfg := factory()
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		TagName:          "json",
		Result:           cfg,
	})
	if err != nil {
		return nil, err
	}
	if err := decoder.Decode(m); err != nil {
		return nil, err
	}
	return cfg, nil
}
