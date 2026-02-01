package activities

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"agent/internal/config/job"
)

// --- WebDAV XML Structures ---

// MultiStatus represents a WebDAV multistatus response.
type MultiStatus struct {
	XMLName   xml.Name   `xml:" multistatus"`
	Responses []Response `xml:"response"`
}

// Response represents a single response in a MultiStatus.
type Response struct {
	Href     string   `xml:"href"`
	PropStat PropStat `xml:"propstat"`
}

// PropStat represents a propstat element.
type PropStat struct {
	Prop   Prop   `xml:"prop"`
	Status string `xml:"status"`
}

// Prop represents a prop element.
type Prop struct {
	ResourceType ResourceType `xml:"resourcetype"`
}

// ResourceType represents a resourcetype element.
type ResourceType struct {
	Collection *struct{} `xml:"collection"`
}

// --- Activity Implementation ---

// DownloadWebDAVActivityInput defines the input for the DownloadWebDAVActivity
type DownloadWebDAVActivityInput struct {
	Job *job.Job
}

// DownloadWebDAVActivityOutput defines the output for the DownloadWebDAVActivity
type DownloadWebDAVActivityOutput struct {
	FilePath string
	Size     int64
	Checksum string
	Name     string
	MimeType string
}

// DownloadWebDAVActivity handles downloading files from a WebDAV server
func (a *Activities) DownloadWebDAVActivity(ctx context.Context, input DownloadWebDAVActivityInput) (*DownloadWebDAVActivityOutput, error) {
	config, ok := input.Job.Config.(*job.WebDAVJobConfig)
	if !ok {
		return nil, fmt.Errorf("invalid job config type: %T", input.Job.Config)
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: config.InsecureSkipVerify},
	}
	client := &http.Client{Transport: tr}

	tmpDir, err := os.MkdirTemp("", "webdav_downloads")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	if err := downloadDir(client, config.URL, tmpDir, config.Username, config.Password); err != nil {
		return nil, err
	}

	// Create a tar.gz archive and calculate checksum simultaneously
	archivePath := filepath.Join(os.TempDir(), fmt.Sprintf("webdav_backup_%s.tar.gz", input.Job.ID))
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return nil, err
	}
	defer archiveFile.Close()

	hasher := sha256.New()
	multiWriter := io.MultiWriter(archiveFile, hasher)

	gw := gzip.NewWriter(multiWriter)
	tw := tar.NewWriter(gw)

	err = filepath.Walk(tmpDir, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(fi, file)
		if err != nil {
			return err
		}

		header.Name = filepath.ToSlash(file[len(tmpDir):])

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !fi.IsDir() {
			data, err := os.Open(file)
			if err != nil {
				return err
			}
			defer data.Close()
			if _, err := io.Copy(tw, data); err != nil {
				return err
			}
		}
		return nil
	})

	// Close writers in the correct order
	tw.Close()
	gw.Close()

	if err != nil {
		return nil, err
	}

	checksum := fmt.Sprintf("%x", hasher.Sum(nil))

	fileInfo, err := archiveFile.Stat()
	if err != nil {
		return nil, err
	}

	return &DownloadWebDAVActivityOutput{
		FilePath: archivePath,
		Size:     fileInfo.Size(),
		Checksum: checksum,
		Name:     filepath.Base(archivePath),
		MimeType: "application/gzip",
	}, nil
}

func downloadDir(client *http.Client, urlString, dest, username, password string) error {
	req, err := http.NewRequest("PROPFIND", urlString, nil)
	if err != nil {
		return err
	}

	req.Header.Add("Depth", "1")
	req.SetBasicAuth(username, password)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMultiStatus {
		return fmt.Errorf("failed to list directory: %s", resp.Status)
	}

	var ms MultiStatus
	if err := xml.NewDecoder(resp.Body).Decode(&ms); err != nil {
		return err
	}

	baseURL, err := url.Parse(urlString)
	if err != nil {
		return err
	}

	for _, res := range ms.Responses {
		hrefURL, err := url.Parse(res.Href)
		if err != nil {
			return err // Or log and continue
		}

		resolvedURL := baseURL.ResolveReference(hrefURL)

		// Skip the root directory itself
		if resolvedURL.String() == baseURL.String() {
			continue
		}

		p := path.Base(resolvedURL.Path)
		destPath := filepath.Join(dest, p)

		if res.PropStat.Status != "HTTP/1.1 200 OK" {
			continue
		}

		isCollection := res.PropStat.Prop.ResourceType.Collection != nil

		if isCollection {
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return err
			}
			if err := downloadDir(client, resolvedURL.String(), destPath, username, password); err != nil {
				return err
			}
		} else {
			if err := downloadFile(client, resolvedURL.String(), destPath, username, password); err != nil {
				return err
			}
		}
	}
	return nil
}

func downloadFile(client *http.Client, urlString, dest, username, password string) error {
	req, err := http.NewRequest("GET", urlString, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(username, password)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: %s", resp.Status)
	}

	file, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}
