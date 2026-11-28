package contracts

import (
	"time"

	"github.com/mark3labs/mcp-go/client"

	"github.com/mozilla-ai/mcpd/internal/domain"
)

// MCPHealthMonitor provides a way to interact with the health status of MCP servers.
// A name is matched by its key, so a lookup does not depend on the case it is given in, and a record
// reports the name the server was configured with.
type MCPHealthMonitor interface {
	// Status returns the health status for a single tracked server.
	Status(name domain.ServerName) (domain.ServerHealth, error)

	// List returns a copy of all known server health records.
	List() []domain.ServerHealth

	// Update records a health check for a tracked server.
	Update(name domain.ServerName, status domain.HealthStatus, latency *time.Duration) error

	// Add registers a new server for health tracking.
	Add(name domain.ServerName)

	// Remove stops tracking health for a server and removes its health data.
	Remove(name domain.ServerName)
}

// MCPClientAccessor provides a way to interact with MCP servers through a client.
// A name is matched by its key, so a lookup does not depend on the case it is given in, and a server is
// reported under the name it was configured with.
type MCPClientAccessor interface {
	// Add registers a client and its tools by server name.
	Add(name domain.ServerName, c client.MCPClient, tools []domain.ToolName)

	// Client returns the client for the given server name.
	// It returns a boolean to indicate whether the client was found.
	Client(name domain.ServerName) (client.MCPClient, bool)

	// Tools returns the allowed tool names for the given server name.
	// It returns a boolean to indicate whether the tools were found.
	Tools(name domain.ServerName) ([]domain.ToolName, bool)

	// UpdateTools updates the tools list for an existing server without restarting the client.
	// Returns an error if the server is not found.
	UpdateTools(name domain.ServerName, tools []domain.ToolName) error

	// List returns all known server names.
	List() []domain.ServerName

	// Remove deletes the client and its tools by server name.
	Remove(name domain.ServerName)
}
