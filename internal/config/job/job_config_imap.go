package job

import "errors"

const (
	JobProviderIMAP JobProvider = "imap"
)

type IMAPExportFormat string

const (
	IMAPExportFormatMbox IMAPExportFormat = "mbox"
	IMAPExportFormatEml  IMAPExportFormat = "eml"
)

type IMAPSecurity string

const (
	IMAPSecurityTLS      IMAPSecurity = "tls"
	IMAPSecurityStartTLS IMAPSecurity = "starttls"
)

type IMAPConfig struct {
	Address      string           `json:"address"`
	Port         int              `json:"port"`
	Username     string           `json:"username"`
	Password     string           `json:"password"`
	Security     IMAPSecurity     `json:"security"`
	ExportFormat IMAPExportFormat `json:"export_format"`
}

func (c *IMAPConfig) Validate() error {
	if c.Address == "" {
		return errors.New("address is required")
	}
	if c.Port == 0 {
		return errors.New("port is required")
	}
	if c.Username == "" {
		return errors.New("username is required")
	}
	if c.Password == "" {
		return errors.New("password is required")
	}
	if c.Security == "" {
		c.Security = IMAPSecurityTLS
	}
	if c.ExportFormat == "" {
		c.ExportFormat = IMAPExportFormatMbox
	}
	return nil
}

func (c *IMAPConfig) Type() JobProvider {
	return JobProviderIMAP
}
