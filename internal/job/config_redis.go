package job

import (
	"errors"

	"github.com/redis/go-redis/v9"
)

const JobProviderRedis Provider = "redis"

type RedisConfig struct {
	ConnectionString string `json:"connection_string"`
}

func (c *RedisConfig) Validate() error {
	if c.ConnectionString == "" {
		return errors.New("connection_string is required")
	}
	if _, err := redis.ParseURL(c.ConnectionString); err != nil {
		return err
	}
	return nil
}

func (c *RedisConfig) Type() Provider { return JobProviderRedis }
