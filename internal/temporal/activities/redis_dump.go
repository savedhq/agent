package activities

import (
	"agent/internal/job"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/redis/go-redis/v9"
	"go.temporal.io/sdk/activity"
)

type RedisDumpActivityInput struct {
	Job *job.Job `json:"job"`
}

type redisKeyEntry struct {
	Key  string `json:"key"`
	TTL  int64  `json:"ttl_ms"` // -1 = no expiry
	Dump string `json:"dump"`   // base64-encoded DUMP payload
}

type redisDumpFile struct {
	Version int             `json:"version"`
	Keys    []redisKeyEntry `json:"keys"`
}

func (a *Activities) RedisDumpActivity(ctx context.Context, input RedisDumpActivityInput) (*DownloadActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("RedisDumpActivity started", "jobId", input.Job.ID)

	cfg, err := job.LoadAs[*job.RedisConfig](*input.Job)
	if err != nil {
		return nil, fmt.Errorf("failed to load Redis config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid Redis config: %w", err)
	}

	opt, err := redis.ParseURL(cfg.ConnectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opt)
	defer client.Close()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	dump := redisDumpFile{Version: 1}

	var cursor uint64
	for {
		keys, nextCursor, err := client.Scan(ctx, cursor, "*", 100).Result()
		if err != nil {
			return nil, fmt.Errorf("SCAN failed: %w", err)
		}

		for _, key := range keys {
			raw, err := client.Dump(ctx, key).Result()
			if err != nil {
				if err == redis.Nil {
					continue
				}
				return nil, fmt.Errorf("DUMP failed for key %q: %w", key, err)
			}

			ttl, err := client.PTTL(ctx, key).Result()
			if err != nil {
				return nil, fmt.Errorf("PTTL failed for key %q: %w", key, err)
			}

			dump.Keys = append(dump.Keys, redisKeyEntry{
				Key:  key,
				TTL:  ttl.Milliseconds(),
				Dump: base64.StdEncoding.EncodeToString([]byte(raw)),
			})
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	filename := fmt.Sprintf("%s.json", input.Job.ID)
	tempFilePath := filepath.Join(a.Config.TempDir, filename)

	f, err := os.Create(tempFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create dump file: %w", err)
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(dump); err != nil {
		return nil, fmt.Errorf("failed to encode dump: %w", err)
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek dump file: %w", err)
	}

	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return nil, fmt.Errorf("failed to calculate hash: %w", err)
	}

	fi, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat dump file: %w", err)
	}

	logger.Info("RedisDumpActivity completed", "filePath", tempFilePath, "size", fi.Size(), "keys", len(dump.Keys))

	return &DownloadActivityOutput{
		FilePath: tempFilePath,
		Size:     fi.Size(),
		Checksum: fmt.Sprintf("%x", hash.Sum(nil)),
		Name:     filename,
		MimeType: "application/json",
	}, nil
}
