package daemon

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/go-hclog"

	"github.com/mozilla-ai/mcpd/v2/internal/runtime"
)

// Dependencies contains required dependencies for the Daemon.
// NewDependencies should be used to create instances of Dependencies.
type Dependencies struct {
	// APIAddr specifies the network address for the APIServer to bind (e.g., "0.0.0.0:8090").
	APIAddr string

	// Logger for daemon and subcomponent (API server) operations.
	Logger hclog.Logger

	// RuntimeServers contains the aggregated runtime servers (config + secrets).
	RuntimeServers []runtime.Server
}

// NewDependencies creates Dependencies with processed runtime servers.
func NewDependencies(
	logger hclog.Logger,
	apiAddr string,
	runtimeServers []runtime.Server,
) (Dependencies, error) {
	if runtimeServers == nil {
		runtimeServers = []runtime.Server{}
	}

	deps := Dependencies{
		APIAddr:        apiAddr,
		Logger:         logger,
		RuntimeServers: runtimeServers,
	}

	if err := deps.Validate(); err != nil {
		return Dependencies{}, err
	}

	return deps, nil
}

// Validate ensures all required dependencies are provided and valid.
func (d Dependencies) Validate() error {
	if d.Logger == nil || reflect.ValueOf(d.Logger).IsNil() {
		return fmt.Errorf("logger cannot be nil")
	}

	if err := validateAddr(d.APIAddr); err != nil {
		return fmt.Errorf("invalid API address '%s': %w", d.APIAddr, err)
	}

	if len(d.RuntimeServers) == 0 {
		return fmt.Errorf("runtime server configurations not found")
	}

	return nil
}
