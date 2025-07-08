package domain

import "time"

const (
	HealthStatusOK          HealthStatus = "ok"
	HealthStatusTimeout     HealthStatus = "timeout"
	HealthStatusUnreachable HealthStatus = "unreachable"
	HealthStatusUnknown     HealthStatus = "unknown"
)

// HealthStatus represents the internal state of an MCP server's availability.
type HealthStatus string

// ServerHealth tracks the internal health state for an MCP server.
type ServerHealth struct {
	Name           string
	Status         HealthStatus
	Latency        *time.Duration
	LastChecked    *time.Time
	LastSuccessful *time.Time
}
