package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

var (
	ErrBadRequest            = errors.New("bad request")
	ErrServerNotFound        = errors.New("server not found")
	ErrToolsNotFound         = errors.New("tools not found")
	ErrToolForbidden         = errors.New("tool not allowed")
	ErrToolListFailed        = errors.New("tool list failed")
	ErrToolCallFailed        = errors.New("tool call failed")
	ErrToolCallFailedUnknown = errors.New("tool call failed (unknown error)")
)

const (
	apiPathPrefix = "/api/v1/"
)

type ApiServer struct {
	clients      map[string]*client.Client
	serverTools  map[string][]string
	clientsMutex *sync.RWMutex
	logger       hclog.Logger
}

func (a *ApiServer) Start(port int, ready chan<- struct{}) error {
	addr := fmt.Sprintf("localhost:%d", port)
	fqdn := fmt.Sprintf("http://%s%sservers", addr, apiPathPrefix) // TODO: HTTP/HTTPS

	mux := http.NewServeMux()
	mux.HandleFunc(apiPathPrefix, a.handleApiRequest)

	fmt.Printf("HTTP REST API listening on: '%s'\n", fqdn)
	a.logger.Info(
		"HTTP REST API listening",
		"address", addr,
		"prefix", apiPathPrefix,
		"endpoint", fqdn,
	)

	// Signal ready just before blocking for serving the API
	close(ready)

	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Printf("HTTP REST API failed to start: %v\n", err)
		a.logger.Error("HTTP REST API failed to start", "error", err)
	}

	return nil
}

func (a *ApiServer) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrBadRequest):
		a.writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrServerNotFound):
		a.writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrToolsNotFound):
		a.writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrToolForbidden):
		a.writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrToolListFailed):
		a.logger.Error("Tool list failed", "error", err)
		a.writeError(w, http.StatusBadGateway, "MCP server error listing tools")
	case errors.Is(err, ErrToolCallFailed):
		a.logger.Error("Tool call failed", "error", err)
		a.writeError(w, http.StatusBadGateway, "MCP server error calling tool")
	case errors.Is(err, ErrToolCallFailedUnknown):
		a.logger.Error("Tool call failed, unknown error", "error", err)
		a.writeError(w, http.StatusBadGateway, "MCP server unknown error calling tool")
	default:
		a.logger.Error("Unexpected error interacting with MCP server", "error", err)
		a.writeError(w, http.StatusInternalServerError, "Internal server error")
	}
}

func (a *ApiServer) writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		a.logger.Error("Error encoding JSON response", "error", err)
	}
}

func (a *ApiServer) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)
	a.writeJSON(w, map[string]string{"error": message})
}

func (a *ApiServer) handleApiRequest(w http.ResponseWriter, r *http.Request) {
	// Trim the prefix and split the path. e.g., /api/v1/servers/time -> ["servers", "time"]
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	// Validate the base path
	if len(parts) == 0 || strings.ToLower(parts[0]) != "servers" {
		a.writeError(w, http.StatusNotFound, "Invalid endpoint. Path must start with /api/v1/servers")
		return
	}

	// TODO: Remove debug logging for this later.
	a.logger.Debug("API request received", "method", r.Method, "path", path)

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second) // TODO: Configure, but can only be as long as the request context timeout
	defer cancel()

	handleResult := func(result any, err error) {
		if err != nil {
			a.handleError(w, err)
		} else {
			// TODO: Remove debug logging for this later.
			a.logger.Debug("API request response", "method", r.Method, "path", path, "result", result)
			a.writeJSON(w, result)
		}
	}

	// Route by path structure and method
	switch {
	case len(parts) == 1 && r.Method == http.MethodGet:
		handleResult(a.listServers(), nil)
		return
	case len(parts) == 2 && r.Method == http.MethodGet:
		result, err := a.listTools(ctx, parts[1])
		handleResult(result, err)
		return
	case len(parts) == 3 && r.Method == http.MethodPost:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			a.handleError(w, fmt.Errorf("%w: failed to read request body: %w", ErrBadRequest, err))
			return
		}
		if len(body) == 0 {
			body = []byte("{}")
		}
		var args map[string]any
		if err := json.Unmarshal(body, &args); err != nil {
			a.handleError(w, fmt.Errorf("%w: invalid JSON arguments: %w", ErrBadRequest, err))
			return
		}
		result, err := a.callTool(ctx, parts[1], parts[2], args)
		handleResult(result, err)
		return

	default:
		a.logger.Warn("Unsupported endpoint requested", "path", r.URL.Path)
		a.writeError(w, http.StatusNotFound, "Unsupported endpoint or method")
		return
	}
}

func (a *ApiServer) listServers() []string {
	a.clientsMutex.RLock()
	defer a.clientsMutex.RUnlock()

	serverNames := make([]string, 0, len(a.clients))

	for name := range a.clients {
		serverNames = append(serverNames, name)
	}

	return serverNames
}

func (a *ApiServer) listTools(ctx context.Context, serverName string) ([]mcp.Tool, error) {
	a.clientsMutex.RLock()
	mcpClient, clientOk := a.clients[serverName]
	allowedTools, toolsOk := a.serverTools[serverName]
	a.clientsMutex.RUnlock()

	if !clientOk {
		return nil, ErrServerNotFound
	}
	if !toolsOk {
		return nil, ErrToolsNotFound
	}

	result, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrToolListFailed, serverName)
	}

	// Only return data on allowed tools.
	var tools []mcp.Tool
	for _, tool := range result.Tools {
		if slices.Contains(allowedTools, tool.Name) {
			tools = append(tools, tool)
		}
	}

	return tools, nil
}

func (a *ApiServer) callTool(ctx context.Context, serverName, toolName string, args map[string]any) (any, error) {
	a.clientsMutex.RLock()
	mcpClient, clientOk := a.clients[serverName]
	allowedTools, toolsOk := a.serverTools[serverName]
	a.clientsMutex.RUnlock()

	if !clientOk {
		return nil, fmt.Errorf("%w: %s", ErrServerNotFound, serverName)
	}

	if !toolsOk || len(allowedTools) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrToolsNotFound, serverName)
	}

	if !slices.Contains(allowedTools, toolName) {
		return nil, fmt.Errorf("%w: %s/%s", ErrToolForbidden, serverName, toolName)
	}

	// TODO: Remove debug logging later.
	a.logger.Debug("Calling tool", "server", serverName, "tool", toolName, "args", args)

	result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %s/%s: %v", ErrToolCallFailed, serverName, toolName, err)
	} else if result.IsError {
		return nil, fmt.Errorf("%w: %s/%s: %v", ErrToolCallFailedUnknown, serverName, toolName, err)
	}

	// The mcp-go library returns a slice of content items. For most tools, this will be a single text item.
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			// We will return the text from the first text content item we find.
			return textContent.Text, nil
		}
	}

	// Fallback to returning the entire content.
	return result.Content, nil // TODO: Is this OK, should we error (also lock down the return type to TextContent)
}
