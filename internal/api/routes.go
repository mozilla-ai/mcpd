package api

import (
	"fmt"
	"net/url"
	"reflect"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/mozilla-ai/mcpd/internal/contracts"
)

// APIVersion is the version used in the OpenAPI spec and URL paths.
const APIVersion = "v1"

// RouteOption configures route behavior.
type RouteOption func(*RouteOptions)

// RouteOptions contains API route behavior that is configured by the daemon.
type RouteOptions struct {
	// ToolCallTimeout bounds how long a single MCP tool call may run.
	ToolCallTimeout time.Duration
}

func newRouteOptions(opts ...RouteOption) RouteOptions {
	options := RouteOptions{
		ToolCallTimeout: DefaultToolCallTimeout(),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}

	return options
}

// DefaultToolCallTimeout returns the default timeout for MCP tool calls.
func DefaultToolCallTimeout() time.Duration {
	return 15 * time.Second
}

// RegisterRoutes registers all API routes on the provided Huma router.
// This is the single source of truth for the API route structure.
// Returns the API path prefix (e.g., "/api/v1") under which the routes are created.
func RegisterRoutes(
	router huma.API,
	healthTracker contracts.MCPHealthMonitor,
	clientManager contracts.MCPClientAccessor,
	opts ...RouteOption,
) (string, error) {
	if router == nil || reflect.ValueOf(router).IsNil() {
		return "", fmt.Errorf("router cannot be nil")
	}
	if clientManager == nil || reflect.ValueOf(clientManager).IsNil() {
		return "", fmt.Errorf("client manager cannot be nil")
	}
	if healthTracker == nil || reflect.ValueOf(healthTracker).IsNil() {
		return "", fmt.Errorf("health tracker cannot be nil")
	}

	routeOptions := newRouteOptions(opts...)

	// Extract API version from the router's OpenAPI spec.
	apiVersionID := router.OpenAPI().Info.Version

	// Safe way to ensure /api/{version}.
	apiPathPrefix, err := url.JoinPath("/api", apiVersionID)
	if err != nil {
		return "", fmt.Errorf("failed to construct API path prefix: %w", err)
	}

	// Group all routes under the /api/{version} prefix.
	versionedGroup := huma.NewGroup(router, apiPathPrefix)
	RegisterHealthRoutes(versionedGroup, healthTracker, "/health")
	RegisterServerRoutes(versionedGroup, clientManager, "/servers", routeOptions)

	return apiPathPrefix, nil
}

// WithToolCallTimeout sets the timeout applied to MCP tool calls.
func WithToolCallTimeout(timeout time.Duration) RouteOption {
	return func(o *RouteOptions) {
		o.ToolCallTimeout = timeout
	}
}
