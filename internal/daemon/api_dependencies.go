package daemon

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/go-hclog"

	"github.com/mozilla-ai/mcpd/internal/contracts"
)

// APIDependencies contains the required external dependencies for the API server.
// NewAPIDependencies should be used to create instances of APIDependencies.
type APIDependencies struct {
	// Addr specifies the network address to bind (e.g., "0.0.0.0:8090").
	Addr string

	// ClientManager handles MCP client connections.
	ClientManager contracts.MCPClientAccessor

	// HealthTracker monitors server health status.
	HealthTracker contracts.MCPHealthMonitor

	// Logger for API server operations.
	Logger hclog.Logger
}

// NewAPIDependencies creates and validates APIDependencies.
func NewAPIDependencies(
	logger hclog.Logger,
	clientManager contracts.MCPClientAccessor,
	healthTracker contracts.MCPHealthMonitor,
	addr string,
) (APIDependencies, error) {
	deps := APIDependencies{
		Addr:          addr,
		ClientManager: clientManager,
		HealthTracker: healthTracker,
		Logger:        logger,
	}

	if err := deps.Validate(); err != nil {
		return APIDependencies{}, err
	}

	return deps, nil
}

// Validate ensures all required dependencies are provided and valid.
func (d APIDependencies) Validate() error {
	if err := validateAddr(d.Addr); err != nil {
		return fmt.Errorf("invalid API address '%s': %w", d.Addr, err)
	}
	if d.ClientManager == nil || reflect.ValueOf(d.ClientManager).IsNil() {
		return fmt.Errorf("client manager cannot be nil")
	}
	if d.HealthTracker == nil || reflect.ValueOf(d.HealthTracker).IsNil() {
		return fmt.Errorf("health tracker cannot be nil")
	}
	if d.Logger == nil || reflect.ValueOf(d.Logger).IsNil() {
		return fmt.Errorf("logger cannot be nil")
	}
	return nil
}
