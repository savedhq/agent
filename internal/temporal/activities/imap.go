package activities

import (
	"agent/internal/config/job"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"archive/zip"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/log"
)

func (a *Activities) IMAPDownloadActivity(ctx context.Context, input DownloadActivityInput) (*DownloadActivityOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("IMAPDownloadActivity started", "jobId", input.Job.ID)

	// 1. Unmarshal IMAP config from the job's provider field
	var imapConfig job.IMAPConfig
	if err := json.Unmarshal([]byte(input.Job.Provider), &imapConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal IMAP config for job %s: %w", input.Job.ID, err)
	}

	// 2. Connect to the IMAP server
	serverAddr := fmt.Sprintf("%s:%d", imapConfig.Address, imapConfig.Port)
	options := imapclient.Options{
		TLSConfig: &tls.Config{ServerName: imapConfig.Address},
	}

	var c *imapclient.Client
	var err error
	if imapConfig.Security == job.IMAPSecurityStartTLS {
		conn, err := net.Dial("tcp", serverAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to dial IMAP server (unencrypted): %w", err)
		}
		c, err = imapclient.NewStartTLS(conn, &options)
		if err != nil {
			return nil, fmt.Errorf("failed to start TLS: %w", err)
		}
	} else { // Default to TLS
		c, err = imapclient.DialTLS(serverAddr, &options)
		if err != nil {
			return nil, fmt.Errorf("failed to dial IMAP server (TLS): %w", err)
		}
	}
	defer c.Close()

	if err := c.Login(imapConfig.Username, imapConfig.Password).Wait(); err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}
	logger.Info("Successfully logged in to IMAP server")

	// 3. Download emails based on export format
	if imapConfig.ExportFormat == job.IMAPExportFormatEml {
		return downloadEml(c, input.Job.ID, logger)
	}
	return downloadMbox(c, input.Job.ID, logger)
}

func downloadMbox(c *imapclient.Client, jobID string, logger log.Logger) (*DownloadActivityOutput, error) {
	// Create a temporary file to store the MBOX backup
	mboxFile, err := os.CreateTemp("", "imap-backup-*.mbox")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer mboxFile.Close()

	// List all mailboxes
	listCmd := c.List("", "*", nil)
	mailboxes, err := listCmd.Collect()
	if err != nil {
		return nil, fmt.Errorf("failed to list mailboxes: %w", err)
	}

	logger.Info("Found mailboxes", "count", len(mailboxes))

	// Iterate through mailboxes and download messages
	for _, mbox := range mailboxes {
		if hasAttr(mbox.Attrs, imap.MailboxAttrNoSelect) {
			logger.Info("Skipping non-selectable mailbox", "mailbox", mbox.Mailbox)
			continue
		}

		selectedMbox, err := c.Select(mbox.Mailbox, nil).Wait()
		if err != nil {
			logger.Error("Failed to select mailbox, skipping", "mailbox", mbox.Mailbox, "error", err)
			continue
		}

		if selectedMbox.NumMessages == 0 {
			logger.Info("No messages in mailbox", "mailbox", mbox.Mailbox)
			continue
		}

		var seqSet imap.SeqSet
		seqSet.AddNum(1, selectedMbox.NumMessages)

		fetchOptions := &imap.FetchOptions{BodySection: []*imap.FetchItemBodySection{{}}}
		fetchCmd := c.Fetch(seqSet, fetchOptions)

		for msg := fetchCmd.Next(); msg != nil; msg = fetchCmd.Next() {
			var body imapclient.FetchItemDataBodySection
			var ok bool
			for item := msg.Next(); item != nil; item = msg.Next() {
				body, ok = item.(imapclient.FetchItemDataBodySection)
				if ok {
					break
				}
			}
			if !ok {
				logger.Error("Failed to get message body", "seqnum", msg.SeqNum)
				continue
			}

			fromLine := fmt.Sprintf("From - %s\n", time.Now().Format(time.UnixDate))
			if _, err := mboxFile.WriteString(fromLine); err != nil {
				return nil, fmt.Errorf("failed to write mbox separator: %w", err)
			}

			if _, err := io.Copy(mboxFile, body.Literal); err != nil {
				return nil, fmt.Errorf("failed to write message to file: %w", err)
			}
			if _, err := mboxFile.WriteString("\n\n"); err != nil {
				return nil, fmt.Errorf("failed to write trailing newlines: %w", err)
			}
		}

		if err := fetchCmd.Close(); err != nil {
			logger.Error("Failed to close fetch command", "mailbox", mbox.Mailbox, "error", err)
		}
	}

	stat, err := mboxFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file stats: %w", err)
	}

	checksum, err := calculateSHA256(mboxFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	logger.Info("IMAP MBOX download completed successfully", "filePath", mboxFile.Name(), "size", stat.Size())

	return &DownloadActivityOutput{
		FilePath: mboxFile.Name(),
		Size:     stat.Size(),
		Checksum: checksum,
		Name:     fmt.Sprintf("imap-backup-%s.mbox", jobID),
		MimeType: "application/mbox",
	}, nil
}

func downloadEml(c *imapclient.Client, jobID string, logger log.Logger) (*DownloadActivityOutput, error) {
	// Create a temporary directory to store the EML files
	emlDir, err := os.MkdirTemp("", "imap-eml-backup-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	// List all mailboxes
	listCmd := c.List("", "*", nil)
	mailboxes, err := listCmd.Collect()
	if err != nil {
		return nil, fmt.Errorf("failed to list mailboxes: %w", err)
	}

	logger.Info("Found mailboxes", "count", len(mailboxes))

	// Iterate through mailboxes and download messages
	for _, mbox := range mailboxes {
		if hasAttr(mbox.Attrs, imap.MailboxAttrNoSelect) {
			logger.Info("Skipping non-selectable mailbox", "mailbox", mbox.Mailbox)
			continue
		}

		selectedMbox, err := c.Select(mbox.Mailbox, nil).Wait()
		if err != nil {
			logger.Error("Failed to select mailbox, skipping", "mailbox", mbox.Mailbox, "error", err)
			continue
		}

		if selectedMbox.NumMessages == 0 {
			logger.Info("No messages in mailbox", "mailbox", mbox.Mailbox)
			continue
		}

		var seqSet imap.SeqSet
		seqSet.AddNum(1, selectedMbox.NumMessages)

		fetchOptions := &imap.FetchOptions{BodySection: []*imap.FetchItemBodySection{{}}}
		fetchCmd := c.Fetch(seqSet, fetchOptions)

		for msg := fetchCmd.Next(); msg != nil; msg = fetchCmd.Next() {
			var body imapclient.FetchItemDataBodySection
			var ok bool
			for item := msg.Next(); item != nil; item = msg.Next() {
				body, ok = item.(imapclient.FetchItemDataBodySection)
				if ok {
					break
				}
			}
			if !ok {
				logger.Error("Failed to get message body", "seqnum", msg.SeqNum)
				continue
			}

			emlFile, err := os.Create(filepath.Join(emlDir, fmt.Sprintf("%d.eml", msg.SeqNum)))
			if err != nil {
				return nil, fmt.Errorf("failed to create eml file: %w", err)
			}
			defer emlFile.Close()

			if _, err := io.Copy(emlFile, body.Literal); err != nil {
				return nil, fmt.Errorf("failed to write message to file: %w", err)
			}
		}

		if err := fetchCmd.Close(); err != nil {
			logger.Error("Failed to close fetch command", "mailbox", mbox.Mailbox, "error", err)
		}
	}

	// Create a zip archive of the EML files
	zipFile, err := os.CreateTemp("", "imap-eml-backup-*.zip")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp zip file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	err = filepath.Walk(emlDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		f, err := zipWriter.Create(info.Name())
		if err != nil {
			return err
		}
		_, err = io.Copy(f, file)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to zip eml files: %w", err)
	}

	stat, err := zipFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file stats: %w", err)
	}

	checksum, err := calculateSHA256(zipFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	logger.Info("IMAP EML download completed successfully", "filePath", zipFile.Name(), "size", stat.Size())

	return &DownloadActivityOutput{
		FilePath: zipFile.Name(),
		Size:     stat.Size(),
		Checksum: checksum,
		Name:     fmt.Sprintf("imap-backup-%s.zip", jobID),
		MimeType: "application/zip",
	}, nil
}

// hasAttr checks if a slice of mailbox attributes contains a specific attribute.
func hasAttr(attrs []imap.MailboxAttr, attr imap.MailboxAttr) bool {
	for _, a := range attrs {
		if a == attr {
			return true
		}
	}
	return false
}

// calculateSHA256 computes the SHA256 checksum of a file.
func calculateSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
