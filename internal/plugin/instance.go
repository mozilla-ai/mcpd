package plugin

import (
	"context"
	"sync"

	"github.com/mozilla-ai/mcpd/v2/internal/config"
)

// Plugin defines the interface for plugin operations.
type Plugin interface {
	// Capabilities returns the flows this plugin supports.
	Capabilities(ctx context.Context) ([]config.Flow, error)

	// CheckHealth verifies the plugin is healthy.
	CheckHealth(ctx context.Context) error

	// CheckReady verifies the plugin is ready to handle requests.
	CheckReady(ctx context.Context) error

	// HandleRequest processes a request.
	HandleRequest(ctx context.Context, req *HTTPRequest) (*HTTPResponse, error)

	// HandleResponse processes a response.
	HandleResponse(ctx context.Context, resp *HTTPResponse) (*HTTPResponse, error)

	// Stop performs graceful shutdown.
	Stop(ctx context.Context) error
}

// HTTPRequest represents an HTTP request passed to plugins.
type HTTPRequest struct {
	// Method is the HTTP method (GET, POST, etc.).
	Method string

	// Path is the request path.
	Path string

	// Headers contains HTTP headers.
	Headers map[string]string

	// Body is the request body.
	Body []byte
}

// HTTPResponse represents an HTTP response from plugins.
type HTTPResponse struct {
	// Continue indicates if the pipeline should continue processing.
	// If false, this response should be returned to the client immediately.
	Continue bool

	// StatusCode is the HTTP status code.
	StatusCode int32

	// Headers contains HTTP headers.
	Headers map[string]string

	// Body is the response body.
	Body []byte

	// ModifiedRequest is the modified request for content transformation plugins.
	// Only valid for content category plugins with CanModify=true.
	ModifiedRequest *HTTPRequest
}

// Instance represents a running plugin instance.
// It wraps a Plugin implementation with metadata and configuration.
type Instance struct {
	Plugin

	name     string
	required bool
	mu       sync.RWMutex

	// capabilities are the flows which the plugin tells us it supports.
	capabilities map[config.Flow]struct{}

	// configuredFlows are the flows the plugin is allowed to be executed for.
	configuredFlows map[config.Flow]struct{}
}

func (i *Instance) IsFlowAllowed(f config.Flow) bool {
	i.mu.RLock()
	defer i.mu.RUnlock()

	_, ok := i.configuredFlows[f]

	return ok
}

// IsFlowSupported checks if the plugin supports the given flow.
// Plugin capabilities are cached on first call to avoid repeated gRPC requests.
func (i *Instance) IsFlowSupported(ctx context.Context, flow config.Flow) (bool, error) {
	i.mu.RLock()
	if i.capabilities != nil {
		_, ok := i.capabilities[flow]
		i.mu.RUnlock()
		return ok, nil
	}
	i.mu.RUnlock()

	// Upgrade to write lock.
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.capabilities == nil {
		flows, err := i.Capabilities(ctx)
		if err != nil {
			return false, err
		}

		i.capabilities = make(map[config.Flow]struct{}, len(flows))
		for _, f := range flows {
			i.capabilities[f] = struct{}{}
		}
	}

	_, ok := i.capabilities[flow]
	return ok, nil
}

// Name returns the plugin's name.
func (i *Instance) Name() string {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return i.name
}

// Required returns whether the plugin is required.
func (i *Instance) Required() bool {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return i.required
}

// SetRequired marks the plugin as required or optional.
func (i *Instance) SetRequired(required bool) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.required = required
}

// SetFlows should be used to set the flows that have been configured for this plugin.
// They are not the same as the flows the plugin reports as its capabilities (i.e. flows the plugin *can* support).
func (i *Instance) SetFlows(flows map[config.Flow]struct{}) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.configuredFlows = flows
}
