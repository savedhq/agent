package job

import (
	"encoding/json"
	"testing"
)

func TestJobMarshalUnmarshal(t *testing.T) {
	// Create a test job with HTTP config
	original := Job{
		ID:       "test-job-1",
		Provider: JobProviderHTTP,
		Config: &HTTPConfig{
			URL:    "https://example.com/file.zip",
			Method: "GET",
			Headers: map[string]string{
				"Authorization": "Bearer token123",
			},
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal job: %v", err)
	}

	t.Logf("Marshaled JSON: %s", string(data))

	// Unmarshal back
	var restored Job
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Failed to unmarshal job: %v", err)
	}

	// Verify basic fields
	if restored.ID != original.ID {
		t.Errorf("ID mismatch: got %s, want %s", restored.ID, original.ID)
	}
	if restored.Provider != original.Provider {
		t.Errorf("Provider mismatch: got %s, want %s", restored.Provider, original.Provider)
	}

	// Verify config type
	httpConfig, err := LoadAs[*HTTPConfig](restored)
	if err != nil {
		t.Fatalf("Failed to load HTTP config: %v", err)
	}

	// Verify config values
	if httpConfig.URL != "https://example.com/file.zip" {
		t.Errorf("URL mismatch: got %s, want %s", httpConfig.URL, "https://example.com/file.zip")
	}
	if httpConfig.Method != "GET" {
		t.Errorf("Method mismatch: got %s, want %s", httpConfig.Method, "GET")
	}
	if httpConfig.Headers["Authorization"] != "Bearer token123" {
		t.Errorf("Header mismatch: got %s, want %s", httpConfig.Headers["Authorization"], "Bearer token123")
	}
}

func TestJobMarshalUnmarshalMultipleProviders(t *testing.T) {
	testCases := []struct {
		name string
		job  Job
	}{
		{
			name: "HTTP",
			job: Job{
				ID:       "http-job",
				Provider: JobProviderHTTP,
				Config: &HTTPConfig{
					URL: "https://example.com",
				},
			},
		},
		{
			name: "Script",
			job: Job{
				ID:       "script-job",
				Provider: JobProviderScript,
				Config: &ScriptConfig{
					Command: "echo hello",
				},
			},
		},
		{
			name: "AWS S3",
			job: Job{
				ID:       "s3-job",
				Provider: JobProviderAWSS3,
				Config: &AWSS3Config{
					Bucket: "my-source-bucket",
					Region: "us-west-2",
					Path:   "backups/data.zip",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Marshal
			data, err := json.Marshal(tc.job)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			// Unmarshal
			var restored Job
			if err := json.Unmarshal(data, &restored); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Verify
			if restored.ID != tc.job.ID {
				t.Errorf("ID mismatch: got %s, want %s", restored.ID, tc.job.ID)
			}
			if restored.Provider != tc.job.Provider {
				t.Errorf("Provider mismatch: got %s, want %s", restored.Provider, tc.job.Provider)
			}
			if restored.Config == nil {
				t.Error("Config is nil after unmarshal")
			}
			if restored.Config.Type() != tc.job.Provider {
				t.Errorf("Config type mismatch: got %s, want %s", restored.Config.Type(), tc.job.Provider)
			}
		})
	}
}

func TestJobMarshalUnmarshalWithEncryptionCompression(t *testing.T) {
	// Create a test job with encryption and compression configs
	original := Job{
		ID:       "test-job-encrypted",
		Provider: JobProviderHTTP,
		Config: &HTTPConfig{
			URL: "https://example.com/file.zip",
		},
		Encryption: EncryptionConfig{
			Enabled:   true,
			PublicKey: "test-public-key-123",
		},
		Compression: CompressionConfig{
			Enabled:   true,
			Algorithm: "gzip",
			Level:     6,
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal job: %v", err)
	}

	t.Logf("Marshaled JSON: %s", string(data))

	// Unmarshal back
	var restored Job
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("Failed to unmarshal job: %v", err)
	}

	// Verify encryption config
	if restored.Encryption.Enabled != original.Encryption.Enabled {
		t.Errorf("Encryption.Enabled mismatch: got %v, want %v", restored.Encryption.Enabled, original.Encryption.Enabled)
	}
	if restored.Encryption.PublicKey != original.Encryption.PublicKey {
		t.Errorf("Encryption.PublicKey mismatch: got %s, want %s", restored.Encryption.PublicKey, original.Encryption.PublicKey)
	}

	// Verify compression config
	if restored.Compression.Enabled != original.Compression.Enabled {
		t.Errorf("Compression.Enabled mismatch: got %v, want %v", restored.Compression.Enabled, original.Compression.Enabled)
	}
	if restored.Compression.Algorithm != original.Compression.Algorithm {
		t.Errorf("Compression.Algorithm mismatch: got %s, want %s", restored.Compression.Algorithm, original.Compression.Algorithm)
	}
	if restored.Compression.Level != original.Compression.Level {
		t.Errorf("Compression.Level mismatch: got %d, want %d", restored.Compression.Level, original.Compression.Level)
	}
}
