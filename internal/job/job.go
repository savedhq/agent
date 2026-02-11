package job

import (
	"encoding/json"
	"fmt"
)

var configFactories = map[Provider]func() Config{
	JobProviderHTTP:        func() Config { return new(HTTPConfig) },
	JobProviderFTP:         func() Config { return new(FTPConfig) },
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

// ConfigFromMap loads a typed Config from a raw map using the provider's factory
func ConfigFromMap(provider Provider, m map[string]any) (Config, error) {
	factory, ok := configFactories[provider]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}

	data, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	cfg := factory()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
