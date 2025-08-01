package api

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/mozilla-ai/mcpd/v2/internal/contracts"
	"github.com/mozilla-ai/mcpd/v2/internal/errors"
)

// ServersResponse represents the wrapped API response for a list of servers.
type ServersResponse struct {
	Body []string
}

// ServerToolsRequest represents the incoming API request for giving the configured tools schemas for a server.
type ServerToolsRequest struct {
	Name string `doc:"Name of the server to lookup tools for" example:"time" path:"name"`
}

// ServerToolCallRequest represents the incoming API request to call a tool on a particular server.
type ServerToolCallRequest struct {
	Server string         `doc:"Name of the server"       example:"time"             path:"server"`
	Tool   string         `doc:"Name of the tool to call" example:"get_current_time" path:"tool"`
	Body   map[string]any `doc:"Body of the tool to call"                            path:"body"`
}

// RegisterServerRoutes sets up health-related API endpoints
func RegisterServerRoutes(routerAPI huma.API, accessor contracts.MCPClientAccessor, apiPathPrefix string) {
	serversAPI := huma.NewGroup(routerAPI, apiPathPrefix)
	tags := []string{"Servers"}

	// Add route at the root of the group (no path specified).
	huma.Register(
		serversAPI,
		huma.Operation{
			OperationID: "listServers",
			Method:      http.MethodGet,
			Summary:     "List all servers",
			Tags:        tags,
		},
		func(ctx context.Context, _ *struct{}) (*ServersResponse, error) {
			return handleServers(accessor)
		},
	)

	huma.Register(
		serversAPI,
		huma.Operation{
			OperationID: "listTools",
			Method:      http.MethodGet,
			Path:        "/{name}/tools",
			Summary:     "List server tools",
			Tags:        append(tags, "Tools"),
		},
		func(ctx context.Context, input *ServerToolsRequest) (*ToolsResponse, error) {
			return handleServerTools(accessor, input.Name)
		},
	)

	huma.Register(
		serversAPI,
		huma.Operation{
			OperationID: "callTool",
			Method:      http.MethodPost,
			Path:        "/{server}/tools/{tool}",
			Summary:     "Call a tool for a server",
			Tags:        append(tags, "Tools"),
		},
		func(ctx context.Context, input *ServerToolCallRequest) (*ToolCallResponse, error) {
			return handleServerToolCall(accessor, input.Server, input.Tool, input.Body)
		},
	)
}

// handleServers returns the list of configured MCP servers.
func handleServers(accessor contracts.MCPClientAccessor) (*ServersResponse, error) {
	servers := accessor.List()
	slices.Sort(servers)

	resp := &ServersResponse{}
	resp.Body = servers

	return resp, nil
}

// handleServerTools returns the schemas for the allowed tools that exist for a given server.
func handleServerTools(accessor contracts.MCPClientAccessor, name string) (*ToolsResponse, error) {
	// TODO: How to get context from Huma/request for the instance of the request without passing it in?
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	mcpClient, clientOk := accessor.Client(name)
	if !clientOk {
		return nil, fmt.Errorf("%w: %s", errors.ErrServerNotFound, name)
	}

	allowedTools, toolsOk := accessor.Tools(name)
	if !toolsOk || len(allowedTools) == 0 {
		return nil, fmt.Errorf("%w: %s", errors.ErrToolsNotFound, name)
	}

	result, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrToolListFailed, name)
	}
	if result == nil {
		return nil, fmt.Errorf("%w: %s: no result", errors.ErrToolListFailed, name)
	}

	// Only return data on allowed tools.
	tools := make([]Tool, 0, len(result.Tools))
	for _, tool := range result.Tools {
		if slices.Contains(allowedTools, tool.Name) {
			data, err := DomainTool(tool).ToAPIType()
			if err != nil {
				return nil, err
			}
			tools = append(tools, data)
		}
	}

	resp := &ToolsResponse{}
	resp.Body = Tools{Tools: tools}

	return resp, nil
}

// handleServerToolCall handles making a call to a specific tool which exists on an MCP server.
func handleServerToolCall(
	accessor contracts.MCPClientAccessor,
	server string,
	tool string,
	data map[string]any,
) (*ToolCallResponse, error) {
	// TODO: How to get context from Huma/request for the instance of the request without passing it in?
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	mcpClient, clientOk := accessor.Client(server)
	if !clientOk {
		return nil, fmt.Errorf("%w: %s", errors.ErrServerNotFound, server)
	}

	allowedTools, toolsOk := accessor.Tools(server)
	if !toolsOk || len(allowedTools) == 0 {
		return nil, fmt.Errorf("%w: %s", errors.ErrToolsNotFound, server)
	}

	if !slices.Contains(allowedTools, tool) {
		return nil, fmt.Errorf("%w: %s/%s", errors.ErrToolForbidden, server, tool)
	}

	result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      tool,
			Arguments: data,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %s/%s: %w", errors.ErrToolCallFailed, server, tool, err)
	} else if result == nil {
		return nil, fmt.Errorf("%w: %s/%s: result was nil", errors.ErrToolCallFailedUnknown, server, tool)
	} else if result.IsError {
		return nil, fmt.Errorf("%w: %s/%s: %v", errors.ErrToolCallFailed, server, tool, extractMessage(result.Content))
	}

	resp := &ToolCallResponse{}
	resp.Body = extractMessage(result.Content)

	return resp, nil
}

// extractMessage attempts to extract a single message from content that is returned from a tool call.
func extractMessage(content []mcp.Content) string {
	message := ""
	if len(content) == 0 {
		return message
	}

	// The mcp-go library returns a slice of content items. For most tools, this will be a single text item.
	for _, c := range content {
		if tc, ok := c.(mcp.TextContent); ok {
			// We will return the text from the first text content item we find.
			return tc.Text
		}
	}

	return message
}
