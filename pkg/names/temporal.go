package names

const (
	WorkflowNameHTTP   = "http"
	WorkflowNameFTP    = "ftp"
	WorkflowNameSFTP   = "sftp"
	WorkflowNameWebDAV = "webdav"
	WorkflowNameNFS    = "nfs"
	// Databases
	WorkflowNameMongoDB    = "mongodb"
	WorkflowNamePostgreSQL = "postgres"
	WorkflowNameMySQL      = "mysql"
	WorkflowNameMSSQL      = "mssql"
	WorkflowNameOracle     = "oracle"
	WorkflowNameRedis      = "redis"
	WorkflowNameCassandra  = "cassandra"
	// Cloud Storage
	WorkflowNameAWSS3       = "aws.s3"
	WorkflowNameAWSDynamoDB = "aws.dynamodb"
	// Email
	WorkflowNameIMAP       = "imap"
	WorkflowNameiCloudMail = "icloud.mail"
	WorkflowNameGmail      = "google.gmail"
	// Productivity
	WorkflowNameiCloudStorage = "icloud.storage"
	WorkflowNameGoogleDrive   = "google.drive"
	WorkflowNameNotion        = "notion"
	// Agent-only
	WorkflowNameScript = "script"
	// System Workflows
	WorkflowNamePostBackup   = "post-backup"
	WorkflowNameCleanup      = "cleanup"
	WorkflowNameUsageReport  = "usage-report"
	WorkflowNameUsageSync    = "usage-sync"
	WorkflowNameArchiveUsage = "archive-usage"

	// Activity Names
	ActivityNameBackupRequest         = "BackupRequestActivity"
	ActivityNameBackupUpload          = "BackupUploadActivity"
	ActivityNameMySQLDump             = "MySQLDumpActivity"
	ActivityNameBackupConfirm         = "BackupConfirmActivity"
	ActivityNameCompressFile          = "CompressFileActivity"
	ActivityNameEncryptFile           = "EncryptFileActivity"
	ActivityNameCreateTempDir         = "CreateTempDirActivity"
	ActivityNameRemoveFile            = "RemoveFileActivity"
	ActivityNameDownload              = "DownloadActivity"
	ActivityNameGetJob                = "GetJobActivity"
	ActivityNameS3Upload              = "S3UploadActivity"
	ActivityNameMoveBackupFile        = "MoveBackupFileActivity"
	ActivityNameListAllJobs           = "ListAllJobsActivity"
	ActivityNameListJobBackups        = "ListJobBackupsActivity"
	ActivityNameDeleteBackupSystem    = "DeleteBackupSystemActivity"
	ActivityNameReportUsageMetrics    = "ReportUsageMetricsActivity"
	ActivityNameCalculateArchiveUsage = "CalculateArchiveUsageActivity"
	ActivityNameCheckLowBalance       = "CheckLowBalanceActivity"
	ActivityNameFileUploadS3          = "FileUploadS3Activity"
)
