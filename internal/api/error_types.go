package api

// ErrorType represents the classification of errors returned via HTTP headers.
type ErrorType string

// HeaderErrorType is the HTTP header key which should be used to convey API error types.
const HeaderErrorType = "Mcpd-Error-Type"

const (
	// PipelineRequestFailure indicates a required plugin or processing step failed during request handling.
	PipelineRequestFailure ErrorType = "request-pipeline-failure"

	// PipelineResponseFailure indicates a required plugin or processing step failed during response handling.
	PipelineResponseFailure ErrorType = "response-pipeline-failure"
)
