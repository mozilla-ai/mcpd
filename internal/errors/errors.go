// Package errors defines domain-level errors used throughout the application.
// These errors represent business logic failures and are mapped to appropriate HTTP status codes at the API boundary.
//
// NOTE: Important for developers
// When adding a new error here, you MUST consider how it should be handled when returned from API endpoints.
//
// Unmapped errors will default to HTTP 500 Internal Server Error.
//
// Don't forget to:
// 1. Add your error to mapError (internal/daemon/api_server.go)
// 2. Add a test case to TestMapError (internal/daemon/api_server_test.go)
// 3. Consider if existing handler tests need updates
package errors

import (
	"errors"
)

var (
	// ErrBadRequest indicates that the client provided invalid input or made a malformed request.
	// This typically results from validation failures or incorrect request parameters.
	// Recommended to map to HTTP 400 Bad Request.
	ErrBadRequest = errors.New("bad request")

	// ErrServerNotFound indicates that the requested MCP server does not exist or is not configured.
	// This occurs when trying to access operations on a server that hasn't been registered.
	// Recommended to map to HTTP 404 Not Found.
	ErrServerNotFound = errors.New("server not found")

	// ErrToolsNotFound indicates that no tools are configured or available for the specified server.
	// This can happen when a server exists but has no tools defined.
	// Recommended to map to HTTP 404 Not Found.
	ErrToolsNotFound = errors.New("tools not found")

	// ErrToolForbidden indicates that the requested tool either does not exist for the MCP server,
	// or exists but is not allowed to be called.
	// This occurs when a tool is not in the server's allowed tools list.
	// Recommended to map to HTTP 403 Forbidden.
	ErrToolForbidden = errors.New("tool not allowed")

	// ErrToolListFailed indicates that listing tools from an MCP server failed.
	// This represents a communication or protocol error with the external MCP server.
	// Recommended to map to HTTP 502 Bad Gateway.
	ErrToolListFailed = errors.New("tool list failed")

	// ErrToolCallFailed indicates that calling a tool on an MCP server failed.
	// This represents a communication or execution error with the external MCP server.
	// Recommended to map to HTTP 502 Bad Gateway.
	ErrToolCallFailed = errors.New("tool call failed")

	// ErrToolCallFailedUnknown indicates that calling a tool failed for an unknown/unexpected reason.
	// This is used when the exact cause of the tool call failure cannot be determined.
	// Recommended to map to HTTP 502 Bad Gateway.
	ErrToolCallFailedUnknown = errors.New("tool call failed (unknown error)")

	// ErrHealthNotTracked indicates that health monitoring is not enabled for the specified server.
	// This occurs when trying to get health status for a server that isn't being monitored.
	// Recommended to map to HTTP 404 Not Found.
	ErrHealthNotTracked = errors.New("server health is not being tracked")

	// ErrPromptListFailed indicates that listing prompts from an MCP server failed.
	// This represents a communication or protocol error with the external MCP server.
	// Recommended to map to HTTP 502 Bad Gateway.
	ErrPromptListFailed = errors.New("prompt list failed")

	// ErrPromptGenerationFailed indicates that getting a prompt from an MCP server failed.
	// This represents a communication or protocol error with the external MCP server.
	// Recommended to map to HTTP 502 Bad Gateway.
	ErrPromptGenerationFailed = errors.New("prompt generation from template failed")

	// ErrPromptNotFound indicates that the requested prompt does not exist.
	// This occurs when trying to get a prompt that hasn't been defined.
	// Recommended to map to HTTP 404 Not Found.
	ErrPromptNotFound = errors.New("prompt not found")

	// ErrPromptForbidden indicates that the requested prompt exists but is not allowed to be accessed.
	// This occurs when a prompt is not in the server's allowed prompts list.
	// Recommended to map to HTTP 403 Forbidden.
	ErrPromptForbidden = errors.New("prompt access forbidden")

	// ErrPromptsNotImplemented indicates that the MCP server does not implement the prompts feature.
	// This occurs when calling prompt methods on servers that only implement tools.
	// Recommended to map to HTTP 501 Not Implemented.
	ErrPromptsNotImplemented = errors.New("prompts not implemented by server")
)
