package daemon

import (
	"context"
	"encoding/json"
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

type ApiServer struct {
	clients      map[string]*client.Client
	serverTools  map[string][]string
	clientsMutex *sync.RWMutex
	logger       hclog.Logger
}

func (a *ApiServer) Start(port int, ready chan<- struct{}) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/", a.handleApiRequest)

	fmt.Println(fmt.Sprintf("HTTP REST API listening on http://localhost:%d/api/v1/servers", port))
	a.logger.Info(fmt.Sprintf("HTTP REST API listening on: http://localhost:%d/api/v1/servers", port))

	// Signal ready just before blocking for serving the API
	close(ready)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), mux); err != nil {
		fmt.Println(fmt.Sprintf("HTTP REST API failed to start: %v", err))
		a.logger.Error("HTTP REST API failed to start", "error", err)
	}

	return nil
}

// handleApiRequest is the main router for all API calls.
func (a *ApiServer) handleApiRequest(w http.ResponseWriter, r *http.Request) {
	a.logger.Debug("API request received", "method", r.Method, "path", r.URL.Path)

	// Trim the prefix and split the path. e.g., /api/v1/servers/time -> ["servers", "time"]
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	// Check for the required "servers" base path.
	if len(parts) == 0 || parts[0] != "servers" {
		http.Error(w, "Invalid endpoint. Path must start with /api/v1/servers", http.StatusNotFound)
		return
	}

	// Route based on the number of parts in the path and the HTTP method.
	switch {
	// GET /api/v1/servers -> List Servers
	case len(parts) == 1 && r.Method == http.MethodGet:
		a.handleListServers(w, r)

	// GET /api/v1/servers/<server-name> -> List Tools
	case len(parts) == 2 && r.Method == http.MethodGet:
		serverName := parts[1]
		a.handleListTools(w, r, serverName)

	// POST /api/v1/servers/<server-name>/<tool-name> -> Call Tool
	case len(parts) == 3 && r.Method == http.MethodPost:
		serverName := parts[1]
		toolName := parts[2]
		a.handleToolCall(w, r, serverName, toolName)

	default:
		a.logger.Warn("Unsupported endpoint requested", "path", r.URL.Path)
		http.Error(w, "Unsupported endpoint or method", http.StatusNotFound)
	}
}

func (a *ApiServer) handleListServers(w http.ResponseWriter, r *http.Request) {
	a.clientsMutex.RLock()
	defer a.clientsMutex.RUnlock()

	serverNames := make([]string, 0, len(a.clients))
	for name := range a.clients {
		serverNames = append(serverNames, name)
	}

	a.writeJSON(w, serverNames)
}

func (a *ApiServer) handleListTools(w http.ResponseWriter, r *http.Request, serverName string) {
	a.clientsMutex.RLock()
	defer a.clientsMutex.RUnlock()

	tools, ok := a.serverTools[serverName]
	if !ok {
		http.Error(w, fmt.Sprintf("Server '%s' not found or has no tool configuration", serverName), http.StatusNotFound)
		return
	}
	a.writeJSON(w, tools)
}

func (a *ApiServer) handleToolCall(w http.ResponseWriter, r *http.Request, serverName, toolName string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		a.logger.Error("Unable to read request body", "error", err)
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	if len(body) == 0 {
		body = []byte("{}")
	}

	a.clientsMutex.RLock()
	mcpClient, clientOk := a.clients[serverName]
	allowedTools, toolsOk := a.serverTools[serverName]
	a.clientsMutex.RUnlock()

	if !clientOk {
		http.Error(w, fmt.Sprintf("Server '%s' not found or has exited", serverName), http.StatusNotFound)
		return
	}

	// If a tools list exists for the server, enforce the allowlist.
	// An empty list means all tools are implicitly allowed, however we pin tools at 'add' time so this shouldn't be empty.
	if toolsOk && len(allowedTools) > 0 && !slices.Contains(allowedTools, toolName) {
		http.Error(w, fmt.Sprintf("Server '%s' does not allow the use of tool '%s'", serverName, toolName), http.StatusForbidden)
		return
	}

	var args map[string]any
	if err := json.Unmarshal(body, &args); err != nil {
		http.Error(w, "Invalid JSON arguments", http.StatusBadRequest)
		return
	}

	a.logger.Info(fmt.Sprintf("[API -> %s] Calling tool '%s'", serverName, toolName), "params", args)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		},
	})
	if err != nil {
		a.logger.Error("Error calling tool", "server", serverName, "tool", toolName, "error", err)
		http.Error(w, "error calling tool", http.StatusInternalServerError)
		return
	}

	// The mcp-go library returns a slice of content items. For most tools, this will be a single text item.
	// We will return the text from the first text content item we find.
	for _, content := range result.Content {
		if textContent, ok := content.(mcp.TextContent); ok {
			// To return raw JSON, we should marshal the text content itself, not just the text string.
			// This preserves the structure e.g., {"type": "text", "text": "Hello, World!"}
			// For a simpler API, we could just return the text. Let's return the full content object.
			a.writeJSON(w, textContent)
			return
		}
	}

	// Fallback for non-text or empty content responses
	a.writeJSON(w, result.Content)
}

func (a *ApiServer) writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// If encoding fails, log it and send a generic server error.
		// This can happen if the value `v` is not serializable.
		a.logger.Error("Error writing JSON response", "error", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
