package internal

const (
	WorkflowNameHTTP   = "http"
	WorkflowNameFTP    = "ftp"
	WorkflowNameSFTP   = "sftp"
	WorkflowNameWebDAV = "webdav"
	WorkflowNameGit    = "git"
	WorkflowNameScript = "script"

	WorkflowNamePostgreSQL = "postgres"
	WorkflowNameMySQL      = "mysql"
	WorkflowNameMSSQL      = "mssql"

	WorkflowNameRedis = "redis"

	WorkflowNameAWSS3       = "aws.s3"
	WorkflowNameAWSDynamoDB = "aws.dynamodb"

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
	ActivityNameFileTransferDownload = "FTPDownloadActivity"
	ActivityNameSFTPDownload         = "SFTPDownloadActivity"

	ActivityNameWebDAVDownload = "WebDAVDownloadActivity"
	ActivityNameGitDownload    = "GitDownloadActivity"
	ActivityNameScriptRun      = "ScriptRunActivity"

	ActivityNameGetJob        = "GetJobActivity"
	ActivityNameFileUploadS3  = "S3UploadActivity"
	ActivityNameFileCleanup   = "FileCleanupActivity"
	ActivityNameCreateTempDir = "CreateTempDirActivity"
	ActivityNameRemoveFile    = "RemoveFileActivity"
)
