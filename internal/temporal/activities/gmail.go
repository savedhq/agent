package activities

import (
	"agent/internal/config/job"
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"go.temporal.io/sdk/activity"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

type GmailExportActivityInput struct {
	Job *job.Job `json:"job"`
}

type GmailExportActivityOutput struct {
	Size     int64  `json:"size"`
	Checksum string `json:"checksum"`
	Name     string `json:"name"`
	MimeType string `json:"mime_type"`
	FilePath string `json:"file_path"`
}

func (a *Activities) GmailExportActivity(ctx context.Context, input GmailExportActivityInput) (*GmailExportActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Debug("GmailExportActivity called", "jobId", input.Job.ID)

	gmailConfig, err := job.LoadAs[*job.GmailConfig](*input.Job)
	if err != nil {
		return nil, fmt.Errorf("failed to load Gmail config: %w", err)
	}

	if err := gmailConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid Gmail config: %w", err)
	}

	// Setup OAuth2
	config := &oauth2.Config{
		ClientID:     gmailConfig.ClientID,
		ClientSecret: gmailConfig.ClientSecret,
		Endpoint:     google.Endpoint,
	}
	token := &oauth2.Token{
		RefreshToken: gmailConfig.RefreshToken,
	}
	tokenSource := config.TokenSource(ctx, token)

	srv, err := gmail.NewService(ctx, option.WithTokenSource(tokenSource))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail service: %w", err)
	}

	// List messages
	var messages []*gmail.Message
	pageToken := ""
	for {
		call := srv.Users.Messages.List("me").Q(gmailConfig.Query)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		resp, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("failed to list messages: %w", err)
		}
		messages = append(messages, resp.Messages...)
		pageToken = resp.NextPageToken
		if pageToken == "" {
			break
		}
		activity.RecordHeartbeat(ctx, fmt.Sprintf("Listed %d messages", len(messages)))
	}

	logger.Info("Found messages", "count", len(messages))

	// Create temp file
	ext := gmailConfig.Format
	if gmailConfig.Format == "eml" {
		ext = "zip"
	}
	filename := fmt.Sprintf("%s-gmail-backup-%s.%s", input.Job.ID, time.Now().Format("20060102-150405"), ext)
	tempPath := filepath.Join(a.Config.TempDir, filename)
	file, err := os.Create(tempPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	var writer io.Writer
	var zipWriter *zip.Writer

	if gmailConfig.Format == "eml" {
		zipWriter = zip.NewWriter(io.MultiWriter(file, hash))
	} else {
		writer = io.MultiWriter(file, hash)
	}

	for i, msgSummary := range messages {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		msg, err := srv.Users.Messages.Get("me", msgSummary.Id).Format("raw").Do()
		if err != nil {
			logger.Warn("Failed to fetch message", "id", msgSummary.Id, "error", err)
			continue
		}

		raw, err := base64.URLEncoding.DecodeString(msg.Raw)
		if err != nil {
			logger.Warn("Failed to decode raw message", "id", msgSummary.Id, "error", err)
			continue
		}

		if gmailConfig.Format == "eml" {
			f, err := zipWriter.Create(fmt.Sprintf("%s.eml", msg.Id))
			if err != nil {
				return nil, fmt.Errorf("failed to create zip entry: %w", err)
			}
			if _, err := f.Write(raw); err != nil {
				return nil, fmt.Errorf("failed to write to zip entry: %w", err)
			}
		} else {
			// MBOX format: From <email> <date>
			date := time.Unix(msg.InternalDate/1000, 0).Format(time.ANSIC)
			header := fmt.Sprintf("From %s %s\n", gmailConfig.Email, date)
			if _, err := writer.Write([]byte(header)); err != nil {
				return nil, fmt.Errorf("failed to write mbox header: %w", err)
			}

			// Mangle "From " at start of lines in body
			lines := bytes.Split(raw, []byte("\n"))
			for j, line := range lines {
				if bytes.HasPrefix(line, []byte("From ")) {
					if _, err := writer.Write([]byte(">")); err != nil {
						return nil, err
					}
				}
				if _, err := writer.Write(line); err != nil {
					return nil, err
				}
				if j < len(lines)-1 {
					if _, err := writer.Write([]byte("\n")); err != nil {
						return nil, err
					}
				}
			}

			if _, err := writer.Write([]byte("\n\n")); err != nil {
				return nil, fmt.Errorf("failed to write mbox separator: %w", err)
			}
		}

		if i%10 == 0 {
			activity.RecordHeartbeat(ctx, fmt.Sprintf("Downloaded %d/%d messages", i+1, len(messages)))
		}
	}

	if zipWriter != nil {
		if err := zipWriter.Close(); err != nil {
			return nil, fmt.Errorf("failed to close zip writer: %w", err)
		}
	}

	// Get final file info for size
	fi, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat temp file: %w", err)
	}

	mimeType := "application/mbox"
	if gmailConfig.Format == "eml" {
		mimeType = "application/zip"
	}

	return &GmailExportActivityOutput{
		FilePath: tempPath,
		Size:     fi.Size(),
		Checksum: fmt.Sprintf("%x", hash.Sum(nil)),
		Name:     filename,
		MimeType: mimeType,
	}, nil
}
