# Agent Temporal Workflows

The agent runs on the customer's infrastructure (on-prem or private cloud). It executes Temporal workflows with **no direct database or S3 access** — instead it calls the backend API over HTTP for job data, backup creation, upload URLs, and confirmation.

## Architecture Overview

```
┌────────────────────────────────────────────────────────────┐
│  Provider Workflow (e.g. HTTPBackupWorkflow)                │
│                                                            │
│  1. GetJobActivity           ← HTTP call to backend API    │
│  2. BackupRequestActivity    ← HTTP call to backend API    │
│  3. Provider-specific download activity (runs locally)     │
│  4. ProcessAndUpload()       ← shared helper               │
│     ├─ CompressFileActivity  (optional, local)             │
│     ├─ EncryptFileActivity   (optional, local)             │
│     ├─ BackupUploadActivity  (gets presigned URL from API) │
│     ├─ S3UploadActivity      (direct upload to S3 via URL) │
│     ├─ FileCleanupActivity   (local temp file removal)     │
│     └─ BackupConfirmActivity (HTTP call to backend API)    │
└────────────────────────────────────────────────────────────┘
```

## Key Differences from Backend

| Aspect                 | Agent                                     | Backend                                    |
|------------------------|-------------------------------------------|--------------------------------------------|
| **DB access**          | None — calls backend API via HTTP         | Direct (repositories injected)             |
| **S3 access**          | Via presigned upload URLs                 | Direct (storage services injected)         |
| **Backup monitoring**  | None — no child workflow                  | `BackupMonitor` + `PostBackupWorkflow`     |
| **Compression/Encrypt**| Handled locally in `ProcessAndUpload`     | Not in workflow (handled elsewhere)        |
| **File cleanup**       | Explicit `FileCleanupActivity` calls      | Managed by worker temp dir                 |
| **Workflow return**    | `error` only                              | `(*GeneralWorkflowOutput, error)`          |
| **Activity struct**    | Config + Auth + HubConfig + TemporalClient| Repos + Storage + Billing + Vault          |

## Key Concepts

### GeneralWorkflowInput

Same structure as backend — all provider workflows accept this:

```go
type GeneralWorkflowInput struct {
    JobId    string `json:"job_id"`
    Provider string `json:"provider"`
}
```

### Flat Sequential Pattern

Unlike the backend's child workflow pattern, the agent uses a simple linear sequence:

1. **GetJob** → fetch job config from backend API
2. **BackupRequest** → tell backend to create a backup record
3. **Download** → provider-specific data acquisition (runs locally)
4. **ProcessAndUpload** → compress → encrypt → get upload URL → upload → cleanup → confirm

There is no `BackupMonitor` or signal-based status tracking. The workflow either succeeds end-to-end or fails.

### ProcessAndUpload (`shared.go`)

This shared function handles all post-download steps:

```go
func ProcessAndUpload(ctx workflow.Context, j *job.Job,
    jobId, backupId, filePath string,
    size int64, checksum, name, mimeType string) error
```

Steps executed:
1. **Compress** (if `job.Compression.Enabled`) → `CompressFileActivity` → cleanup original
2. **Encrypt** (if `job.Encryption.Enabled`) → `EncryptFileActivity` → cleanup previous
3. **BackupUpload** → calls backend API to get a presigned S3 upload URL
4. **S3Upload** → uploads file directly to S3 using the presigned URL
5. **Cleanup** → removes the local temp file
6. **BackupConfirm** → calls backend API to mark backup as completed

### Activities Struct

Agent activities use API-based communication:

```go
type Activities struct {
    Config         *config.Config           // paths, temp dir
    Auth           authentication.AuthService // API auth tokens
    Hub            *config.HubConfig        // backend API base URL
    TemporalClient client.Client            // for starting child workflows if needed
}
```

### Activity Options

Same defaults as backend:

```go
workflow.ActivityOptions{
    StartToCloseTimeout: 30 * time.Minute,
    RetryPolicy: &temporal.RetryPolicy{
        InitialInterval:    time.Second,
        BackoffCoefficient: 2.0,
        MaximumInterval:    5 * time.Minute,
        MaximumAttempts:    3,
    },
}
```

## Adding a New Provider Workflow

### Step 1: Register Constants

Add to `internal/constants.go`:

```go
ActivityNameMyProviderDownload = "MyProviderDownloadActivity"
```

Workflow names are defined alongside existing ones (must match provider string from job config).

### Step 2: Create the Activity

Create `activities/my_provider_download.go`:

```go
type MyProviderDownloadActivityInput struct {
    Job *job.Job `json:"job"`
}

type MyProviderDownloadActivityOutput = DownloadActivityOutput  // reuse if same shape

func (a *Activities) MyProviderDownloadActivity(ctx context.Context, input MyProviderDownloadActivityInput) (*DownloadActivityOutput, error) {
    // 1. Load typed config: job.LoadAs[*job.MyProviderConfig](*input.Job)
    // 2. Validate config
    // 3. Create temp dir: os.MkdirAll(a.Config.TempDir, 0o755)
    // 4. Download/dump data to temp file
    // 5. Calculate sha256 checksum and file size
    // 6. Return DownloadActivityOutput{FilePath, Size, Checksum, Name, MimeType}
}
```

**Note:** Agent activities do NOT have direct DB or S3 access. If you need secrets, they come from the job config (fetched via `GetJobActivity`).

### Step 3: Create the Workflow

Create `workflows/my_provider.go` following the flat pattern used by HTTP, FTP, and Git:

```go
func MyProviderBackupWorkflow(ctx workflow.Context, input GeneralWorkflowInput) error {
    logger := workflow.GetLogger(ctx)
    logger.Info("MyProviderBackupWorkflow started", "jobId", input.JobId)

    // 1. Set activity options (copy standard options)
    ctx = workflow.WithActivityOptions(ctx, ...)

    // 2. Get job from backend API
    var getJobOut activities.GetJobActivityOutput
    workflow.ExecuteActivity(ctx, internal.ActivityNameGetJob,
        activities.GetJobActivityInput{JobId: input.JobId}).Get(ctx, &getJobOut)

    // 3. Create backup record via backend API
    var backupOut activities.BackupRequestActivityOutput
    workflow.ExecuteActivity(ctx, internal.ActivityNameBackupRequest,
        activities.BackupRequestActivityInput{Job: getJobOut.Job}).Get(ctx, &backupOut)

    // 4. Provider-specific download (runs locally)
    var dlOut activities.DownloadActivityOutput
    workflow.ExecuteActivity(ctx, internal.ActivityNameMyProviderDownload,
        activities.MyProviderDownloadActivityInput{Job: getJobOut.Job}).Get(ctx, &dlOut)

    // 5. Compress → encrypt → upload → confirm (shared helper)
    return ProcessAndUpload(ctx, getJobOut.Job, input.JobId, backupOut.ID.String(),
        dlOut.FilePath, dlOut.Size, dlOut.Checksum, dlOut.Name, dlOut.MimeType)
}
```

### Step 4: Register the Workflow

Register the workflow with the Temporal worker, mapping the workflow name to the function.

## Existing Providers

| Provider      | Workflow File        | Download Activity              | Status  |
|---------------|----------------------|--------------------------------|---------|
| HTTP          | `http.go`            | `DownloadActivity`             | Tested  |
| FTP           | `ftp.go`             | `FileTransferDownloadActivity` | Tested  |
| SFTP          | `sftp.go`            | `SFTPDownloadActivity`         | Untested|
| Git           | `git.go`             | `GitDownloadActivity`          | Tested  |
| WebDAV        | `webdav.go`          | `WebDAVDownloadActivity`       | Untested|
| PostgreSQL    | `postgresql.go`      | `PostgreSQLDumpActivity`       | Untested|
| MySQL         | `mysql.go`           | `MySQLDumpActivity`            | Untested|
| MSSQL         | `mssql.go`           | `MSSQLConnectActivity` + `MSSQLDumpActivity` | Untested|
| Redis         | `redis.go`           | `RedisDumpActivity`            | Untested|
| AWS S3        | `aws_s3.go`          | `AWSS3DownloadActivity`        | Untested|
| AWS DynamoDB  | `aws_dynamodb.go`    | `AWSDynamoDBDumpActivity`      | Untested|
| Script        | `script.go`          | `ScriptRunActivity`            | Untested|
