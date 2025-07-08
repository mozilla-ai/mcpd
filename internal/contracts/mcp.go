package contracts

import (
	"time"

	"github.com/mark3labs/mcp-go/client"

	"github.com/mozilla-ai/mcpd/v2/internal/domain"
)

// MCPHealthMonitor provides a way to interact with the health status of MCP servers.
type MCPHealthMonitor interface {
	// Status returns the health status for a single tracked server.
	Status(name string) (domain.ServerHealth, error)

	// List returns a copy of all known server health records.
	List() []domain.ServerHealth

	// Update records a health check for a tracked server.
	Update(name string, status domain.HealthStatus, latency *time.Duration) error
}

// MCPClientAccessor provides a way to interact with MCP servers through a client.
type MCPClientAccessor interface {
	// Add registers a client and its tools by server name.
	Add(name string, c *client.Client, tools []string)

	// Client returns the client for the given server name.
	// It returns a boolean to indicate whether the client was found.
	Client(name string) (*client.Client, bool)

	// Tools returns the tools for the given server name.
	// It returns a boolean to indicate whether the tools were found.
	Tools(name string) ([]string, bool)

	// List returns all known server names.
	List() []string

	// Remove deletes the client and its tools by server name.
	Remove(name string)
}
