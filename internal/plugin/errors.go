package plugin

import "errors"

// ErrRequiredPluginFailed is returned when a required plugin fails to execute.
// This signals that the request/response should be rejected.
var ErrRequiredPluginFailed = errors.New("required plugin failed")
