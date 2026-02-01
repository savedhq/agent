package workflows

// GeneralWorkflowInput defines the standardized input for all job workflows
// This is sent by Temporal when triggering a workflow
type GeneralWorkflowInput struct {
	JobId            string `json:"job_id"`
	Provider         string `json:"provider"`
	CompressionLevel int    `json:"compression_level"`
}
