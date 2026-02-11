package internal

const (
	WorkflowNameHTTP   = "http"
	WorkflowNameFTP    = "ftp"
	WorkflowNameWebDAV = "webdav"
	WorkflowNameGit    = "git"
	WorkflowNameScript = "script"

	WorkflowNamePostgreSQL = "postgres"
	WorkflowNameMySQL      = "mysql"
	WorkflowNameMSSQL      = "mssql"

	WorkflowNameRedis = "redis"

	WorkflowNameAWSS3       = "aws.s3"
	WorkflowNameAWSDynamoDB = "aws.dynamodb"

	WorkflowNamePostBackup   = "post-backup"
	WorkflowNameCleanup      = "cleanup"
	WorkflowNameUsageReport  = "usage-report"
	WorkflowNameUsageSync    = "usage-sync"
	WorkflowNameArchiveUsage = "archive-usage"

	ActivityNameBackupRequest = "BackupRequestActivity"
	ActivityNameBackupUpload  = "BackupUploadActivity"
	ActivityNameBackupConfirm = "BackupConfirmActivity"
	ActivityNameCompressFile  = "CompressFileActivity"
	ActivityNameEncryptFile   = "EncryptFileActivity"
	ActivityNameDownload      = "DownloadActivity"

	ActivityNameMSSQLConnect         = "MSSQLConnectActivity"
	ActivityNameMSSQLDump            = "MSSQLDumpActivity"
	ActivityNameMySQLDump            = "MySQLDumpActivity"
	ActivityNamePostgreSQLDump       = "PostgreSQLDumpActivity"
	ActivityNameRedisDump            = "RedisDumpActivity"
	ActivityNameAWSDynamoDBDump      = "AWSDynamoDBDumpActivity"
	ActivityNameAWSS3Download        = "AWSS3DownloadActivity"
	ActivityNameFileTransferDownload = "FileTransferDownloadActivity"

	ActivityNameWebDAVDownload = "WebDAVDownloadActivity"
	ActivityNameGitDownload    = "GitDownloadActivity"
	ActivityNameScriptRun      = "ScriptRunActivity"

	ActivityNameGetJob                = "GetJobActivity"
	ActivityNameFileUploadS3          = "S3UploadActivity"
	ActivityNameMoveBackupFile        = "MoveBackupFileActivity"
	ActivityNameListAllJobs           = "ListAllJobsActivity"
	ActivityNameListJobBackups        = "ListJobBackupsActivity"
	ActivityNameDeleteBackupSystem    = "DeleteBackupSystemActivity"
	ActivityNameReportUsageMetrics    = "ReportUsageMetricsActivity"
	ActivityNameCalculateArchiveUsage = "CalculateArchiveUsageActivity"
	ActivityNameCheckLowBalance       = "CheckLowBalanceActivity"
	ActivityNameFileCleanup           = "FileCleanupActivity"
	ActivityNameCreateTempDir         = "CreateTempDirActivity"
	ActivityNameRemoveFile            = "RemoveFileActivity"
)
