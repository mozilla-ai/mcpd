package api

import (
	"context"
	"fmt"
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
	LastChecked    *time.Time   `json:"lastChecked,omitempty"`
	LastSuccessful *time.Time   `json:"lastSuccessful,omitempty"`
}

// ServersHealth represents a collection of ServerHealth.
type ServersHealth struct {
	Servers []ServerHealth `json:"servers"`
}

// ServersHealthResponse is the response for GET /health
type ServersHealthResponse struct {
	Body struct {
		Servers []ServerHealth `doc:"Tracked MCP server health statuses" json:"servers"`
	}
}

// ServerHealthRequest represents the incoming request for obtaining ServerHealth.
type ServerHealthRequest struct {
	Name string `doc:"Name of the server to check" example:"time" path:"name"`
}

// ServerHealthResponse represents the wrapped API response for a ServerHealth.
type ServerHealthResponse struct {
	Body ServerHealth
}

// ToAPIType can be used to convert a wrapped domain type to an API-safe type.
func (d DomainServerHealth) ToAPIType() (ServerHealth, error) {
	status, err := parseHealthStatus(d.Status)
	if err != nil {
		return ServerHealth{}, err
	}

	var latency *string
	if d.Latency != nil {
		s := d.Latency.String()
		latency = &s
	}
	return ServerHealth{
		Name:           d.Name,
		Status:         status,
		Latency:        latency,
		LastChecked:    d.LastChecked,
		LastSuccessful: d.LastSuccessful,
	}, nil
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
		data, err := DomainServerHealth(s).ToAPIType()
		if err != nil {
			return nil, err
		}
		apiServers = append(apiServers, data)
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

	data, err := DomainServerHealth(health).ToAPIType()
	if err != nil {
		return nil, err
	}

	response := ServerHealthResponse{}
	response.Body = data

	return &response, nil
}

func parseHealthStatus(status domain.HealthStatus) (HealthStatus, error) {
	switch status {
	case domain.HealthStatusOK:
		return HealthStatusOK, nil
	case domain.HealthStatusTimeout:
		return HealthStatusTimeout, nil
	case domain.HealthStatusUnreachable:
		return HealthStatusUnreachable, nil
	case domain.HealthStatusUnknown:
		return HealthStatusUnknown, nil
	default:
		return "", fmt.Errorf("unknown health status: %s", status)
	}
}
