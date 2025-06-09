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

func (a *ApiServer) Start(port int) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/", a.handleApiRequest)
	mux.HandleFunc("/api/v1", a.handleApiRequest)

	// log.Println(fmt.Sprintf("HTTP REST API listening on :%d", port))
	a.logger.Info(fmt.Sprintf("HTTP REST API listening on :%d", port))

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), mux); err != nil {
		// log.Fatalf("API daemon failed: %v", err)
		a.logger.Error("HTTP REST API failed to start", "error", err)
	}

	return nil
}

func (a *ApiServer) handleApiRequest(w http.ResponseWriter, r *http.Request) {
	a.logger.Debug("API request received", "method", r.Method, "path", r.URL.Path)

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 || parts[0] != "api" || parts[1] != "v1" {
		http.Error(w, "Invalid API prefix", http.StatusBadRequest)
		return
	}

	switch {
	case len(parts) == 2 && r.Method == http.MethodGet:
		a.handleListServers(w, r)
	case len(parts) == 3 && r.Method == http.MethodGet:
		a.handleListTools(w, r, parts[2])
	case len(parts) == 4 && r.Method == http.MethodPost:
		a.handleToolCall(w, r, parts[2], parts[3])
	default:
		http.Error(w, "Unsupported endpoint", http.StatusNotFound)
	}
}

func (a *ApiServer) handleListServers(w http.ResponseWriter, r *http.Request) {
	serverNames := make([]string, 0, len(a.clients))
	for name := range a.clients {
		serverNames = append(serverNames, name)
	}

	writeJSON(w, serverNames)
}

func (a *ApiServer) handleListTools(w http.ResponseWriter, r *http.Request, serverName string) {
	a.clientsMutex.Lock()
	defer a.clientsMutex.Unlock()

	// TODO: This will only list tools that have been explicitly allowed, we don't know all the tools 'available'.
	tools, ok := a.serverTools[serverName]
	if !ok {
		http.Error(w, fmt.Sprintf("Tools for server '%s' not found", serverName), http.StatusNotFound)
		return
	}
	writeJSON(w, tools)
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

	a.clientsMutex.Lock()
	mcpClient, ok := a.clients[serverName]
	a.clientsMutex.Unlock()
	if !ok {
		http.Error(w, fmt.Sprintf("Server '%s' not found or has exited", serverName), http.StatusNotFound)
		return
	}

	if tools, ok := a.serverTools[serverName]; ok && !slices.Contains(tools, toolName) {
		http.Error(w, fmt.Sprintf("Server '%s' does not allow the use of tool '%s'", serverName, toolName), http.StatusNotFound)
		return
	}

	var args map[string]any
	if err := json.Unmarshal(body, &args); err != nil {
		http.Error(w, "Invalid arguments", http.StatusBadRequest)
		return
	}

	a.logger.Info(fmt.Sprintf("[API -> %s] Calling tool '%s' with params: %v", serverName, toolName, args))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		},
	})
	if err != nil || result.IsError {
		http.Error(w, "error calling tool", http.StatusInternalServerError)
		return
	}

	var responses []string
	for _, content := range result.Content {
		textContent, ok := content.(mcp.TextContent)
		if !ok {
			http.Error(w, "error parsing response content", http.StatusInternalServerError)
			return
		}
		responses = append(responses, textContent.Text)
	}

	writeJSON(w, responses)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
