package api

import (
	"context"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/mozilla-ai/mcpd/v2/internal/contracts"
	"github.com/mozilla-ai/mcpd/v2/internal/domain"
)

const (
	HealthStatusOK          HealthStatus = "ok"
	HealthStatusTimeout     HealthStatus = "timeout"
	HealthStatusUnreachable HealthStatus = "unreachable"
	HealthStatusUnknown     HealthStatus = "unknown"
)

// DomainServerHealth is a wrapper that allows receivers to be declared in the API package that deal with domain types.
type DomainServerHealth domain.ServerHealth

// HealthStatus represents the current status of a particular MCP server when establishing its health.
type HealthStatus string

// ServerHealth is used to provide information about ongoing health checks that are performed on running MCP servers.
type ServerHealth struct {
	Name           string       `json:"name"`
	Status         HealthStatus `json:"status"`
	Latency        *string      `json:"latency,omitempty"`
	LastChecked    *time.Time   `json:"last_checked,omitempty"`
	LastSuccessful *time.Time   `json:"last_successful,omitempty"`
}

// ServersHealth represents a collection of ServerHealth.
type ServersHealth struct {
	Servers []ServerHealth `json:"servers"`
}

// ServersHealthResponse is the response for GET /health
type ServersHealthResponse struct {
	Body struct {
		Servers []ServerHealth `json:"servers" doc:"Tracked MCP server health statuses"`
	}
}

// ServerHealthRequest represents the incoming request for obtaining ServerHealth.
type ServerHealthRequest struct {
	Name string `path:"name" example:"time" doc:"Name of the server to check"`
}

// ServerHealthResponse represents the wrapped API response for a ServerHealth.
type ServerHealthResponse struct {
	Body ServerHealth
}

// ToAPIType can be used to convert a wrapped domain type to an API-safe type.
func (d DomainServerHealth) ToAPIType() ServerHealth {
	var latency *string
	if d.Latency != nil {
		s := d.Latency.String()
		latency = &s
	}
	return ServerHealth{
		Name:           d.Name,
		Status:         HealthStatus(d.Status), // TODO: Validation?
		Latency:        latency,
		LastChecked:    d.LastChecked,
		LastSuccessful: d.LastSuccessful,
	}
}

// RegisterHealthRoutes sets up health-related API endpoint routes.
func RegisterHealthRoutes(routerAPI huma.API, monitor contracts.MCPHealthMonitor, apiPathPrefix string) {
	healthAPI := huma.NewGroup(routerAPI, apiPathPrefix)
	tags := []string{"Health"}

	huma.Register(
		healthAPI,
		huma.Operation{
			OperationID: "listServersHealth",
			Method:      http.MethodGet,
			Path:        "/servers",
			Summary:     "List the health statuses for all servers",
			Tags:        tags,
		},
		func(ctx context.Context, _ *struct{}) (*ServersHealthResponse, error) {
			return handleHealthServers(monitor)
		},
	)

	huma.Register(
		healthAPI,
		huma.Operation{
			OperationID: "getServerHealth",
			Method:      http.MethodGet,
			Path:        "/servers/{name}",
			Summary:     "Get the health status of a server",
			Tags:        tags,
		},
		func(ctx context.Context, input *ServerHealthRequest) (*ServerHealthResponse, error) {
			return handleHealthServer(monitor, input.Name)
		},
	)
}

// handleHealthServers is the handler for retrieving the current health for all registered MCP servers.
func handleHealthServers(monitor contracts.MCPHealthMonitor) (*ServersHealthResponse, error) {
	servers := monitor.List()

	slices.SortFunc(servers, func(a, b domain.ServerHealth) int {
		return strings.Compare(a.Name, b.Name)
	})

	apiServers := make([]ServerHealth, 0, len(servers))
	for _, s := range servers {
		apiServers = append(apiServers, DomainServerHealth(s).ToAPIType())
	}

	resp := &ServersHealthResponse{}
	resp.Body.Servers = apiServers

	return resp, nil
}

// handleHealthServer is the handler for retrieving the current health the specified registered MCP server.
func handleHealthServer(monitor contracts.MCPHealthMonitor, name string) (*ServerHealthResponse, error) {
	health, err := monitor.Status(name)
	if err != nil {
		return nil, err
	}

	response := ServerHealthResponse{}
	response.Body = DomainServerHealth(health).ToAPIType()

	return &response, nil
}
