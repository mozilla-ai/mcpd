package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

type ApiServer struct {
	clients      map[string]*client.Client
	clientsMutex *sync.Mutex
	logger       hclog.Logger
}

func (a *ApiServer) Start(port int) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/", a.handleApiRequest)

	// log.Println(fmt.Sprintf("HTTP REST API listening on :%d", port))
	a.logger.Info(fmt.Sprintf("HTTP REST API listening on :%d", port))

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), mux); err != nil {
		// log.Fatalf("API daemon failed: %v", err)
		a.logger.Error("HTTP REST API failed to start", "error", err)
	}

	return nil
}

// handleApiRequest now uses the client's Call method.
func (a *ApiServer) handleApiRequest(w http.ResponseWriter, r *http.Request) {
	a.logger.Debug("API request received", "method", r.Method, "path", r.URL.Path)

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 4 {
		a.logger.Error("API request has invalid path, expected /api/v1/{daemon}/{tool}", "path", r.URL.Path)
		http.Error(w, "Invalid URL format. Expected /api/v1/{daemon}/{tool}", http.StatusBadRequest)
		return
	}
	serverName := parts[2]
	toolName := parts[3]

	body, err := io.ReadAll(r.Body)
	if err != nil {
		a.logger.Error("Unable to read request body", "method", r.Method, "path", r.URL.Path, "error", err)
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

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var args map[string]any
	if err := json.Unmarshal(body, &args); err != nil {
		http.Error(w, "Invalid arguments", http.StatusBadRequest)
		return
	}

	a.logger.Info(fmt.Sprintf("[API -> %s] Calling tool '%s' with params: %v", serverName, toolName, args))

	result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		},
	})
	if err != nil {
		a.logger.Error("Failed to call tool", "error", err)
		http.Error(w, "Failed to call tool", http.StatusInternalServerError)
		return
	}

	a.logger.Info(fmt.Sprintf("[API <- %s] Received result: %#v", serverName, result))

	if result.IsError {
		a.logger.Error(fmt.Sprintf("Server '%s' failed to call tool '%s'", serverName, toolName))
		http.Error(w, "error calling tool", http.StatusInternalServerError)
		return
	}

	var responses []string
	for _, content := range result.Content {
		textContent, ok := content.(mcp.TextContent)
		if !ok {
			a.logger.Error(fmt.Sprintf("Server '%s' tool '%s' failed to parse content (as text): '%#v'", serverName, toolName, content))
			http.Error(w, "error parsing response content", http.StatusInternalServerError)
			return
		}

		responses = append(responses, textContent.Text)
		a.logger.Info(fmt.Sprintf("[API <- %s] Received content: %s from tool: %s", serverName, textContent, toolName))
	}

	jsonBytes, err := json.Marshal(responses)
	if err != nil {
		http.Error(w, "error marshaling json", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonBytes)
}
