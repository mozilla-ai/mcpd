package api

import (
	"fmt"
	"net/url"
	"reflect"

	"github.com/danielgtaylor/huma/v2"

	"github.com/mozilla-ai/mcpd/v2/internal/contracts"
)

// APIVersion is the version used in the OpenAPI spec and URL paths.
const APIVersion = "v1"

// RegisterRoutes registers all API routes on the provided Huma router.
// This is the single source of truth for the API route structure.
// Returns the API path prefix (e.g., "/api/v1") under which the routes are created.
func RegisterRoutes(
	router huma.API,
	healthTracker contracts.MCPHealthMonitor,
	clientManager contracts.MCPClientAccessor,
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
	RegisterServerRoutes(versionedGroup, clientManager, "/servers")

	return apiPathPrefix, nil
}
