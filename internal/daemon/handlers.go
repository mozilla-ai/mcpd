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
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mark3labs/mcp-go/mcp"
)

// TODO: support 'options' for ApiServer that contain timeout, then remove duplicate magic numbers for timeouts.

// handleHealthServers handles GET requests to the /health/servers endpoint.
//
// It responds with a JSON object containing the health status of all tracked MCP servers.
// Each server's health includes status, latency, and last checked timestamps.
//
// @Summary Get health status for all MCP servers
// @Description Returns the latest health information for all tracked MCP servers.
// @Tags health
// @Produce json
// @Success 200 {object} map[string][]daemon.ServerHealth
// @Router /api/v1/health/servers [get]
func (a *ApiServer) handleHealthServers(w http.ResponseWriter, _ *http.Request) {
	servers := a.healthTracker.List()

	slices.SortFunc(servers, func(a, b ServerHealth) int {
		return strings.Compare(a.Name, b.Name)
	})

	response := map[string]any{
		"servers": servers,
	}

	a.writeJSON(w, response)
}

// handleHealthServer handles GET requests to the /health/servers/{server} endpoint.
//
// It responds with a JSON object describing the health of the specified MCP server,
// or a 404 error if the server name is not known.
//
// @Summary Get health status for a specific MCP server
// @Description Returns the latest health information for a single tracked MCP server.
// @Tags health
// @Produce json
// @Param server path string true "Server name"
// @Success 200 {object} daemon.ServerHealth
// @Failure 404 {string} string "Server not found"
// @Router /api/v1/health/servers/{server} [get]
func (a *ApiServer) handleHealthServer(w http.ResponseWriter, r *http.Request) {
	server := chi.URLParam(r, "server")

	health, err := a.healthTracker.Status(server)
	if err != nil {
		http.Error(w, fmt.Sprintf("server not found: %s", server), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(health)
}

// serversHandler handles GET requests to the /servers endpoint.
//
// It responds with a JSON array of MCP server names configured in the system.
// This endpoint is useful for health checks, discovery, or introspection.
//
// @Summary List available MCP servers
// @Description Returns the names of all currently configured MCP servers.
// @Tags servers
// @Produce json
// @Success 200 {array} string
// @Router /api/v1/servers [get]
func (a *ApiServer) serversHandler(w http.ResponseWriter, _ *http.Request) {
	servers := a.clientManager.List()
	slices.Sort(servers)
	a.writeJSON(w, servers)
}

// handleServer handles GET requests to the /servers/{server} endpoint.
//
// It returns the list of tools available on the specified MCP server.
//
// @Summary List tools for a specific MCP server
// @Description Returns the available tools on the given server.
// @Tags servers
// @Produce json
// @Param server path string true "Server name"
// @Success 200 {array} mcp.Tool
// @Failure 404 {object} map[string]string
// @Failure 502 {object} map[string]string
// @Router /api/v1/servers/{server} [get]
func (a *ApiServer) handleServer(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	server := chi.URLParam(r, "server")

	mcpClient, clientOk := a.clientManager.Client(server)
	if !clientOk {
		a.handleError(w, fmt.Errorf("%w: %s", ErrServerNotFound, server))
		return
	}

	allowedTools, toolsOk := a.clientManager.Tools(server)
	if !toolsOk || len(allowedTools) == 0 {
		a.handleError(w, fmt.Errorf("%w: %s", ErrToolsNotFound, server))
		return
	}

	result, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		a.handleError(w, fmt.Errorf("%w: %s", ErrToolListFailed, server))
		return
	}
	if result == nil {
		a.handleError(w, fmt.Errorf("%w: %s: no result", ErrToolListFailed, server))
		return
	}

	// Only return data on allowed tools.
	var tools []mcp.Tool
	for _, tool := range result.Tools {
		if slices.Contains(allowedTools, tool.Name) {
			tools = append(tools, tool)
		}
	}

	a.writeJSON(w, tools)
}

// handleServerCallTool handles POST requests to the /servers/{server}/{tool} endpoint.
//
// It invokes a specific tool on the given MCP server, passing optional JSON arguments.
//
// @Summary Call a tool on a server
// @Description Executes the specified tool with optional input arguments.
// @Tags servers
// @Accept json
// @Produce json
// @Param server path string true "Server name"
// @Param tool path string true "Tool name"
// @Param args body map[string]any false "Tool arguments"
// @Success 200 {object} any
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 502 {object} map[string]string
// @Router /api/v1/servers/{server}/{tool} [post]
func (a *ApiServer) handleServerCallTool(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	server := chi.URLParam(r, "server")
	tool := chi.URLParam(r, "tool")

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

	result, err := a.callTool(ctx, server, tool, args)
	a.handleResult(w, result, err)
}

func (a *ApiServer) handleResult(w http.ResponseWriter, result any, err error) {
	if err != nil {
		a.handleError(w, err)
	} else {
		a.writeJSON(w, result)
	}
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
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		a.logger.Error("Error encoding JSON response", "error", err)
	}
}

func (a *ApiServer) writeError(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)
	a.writeJSON(w, map[string]string{"error": message})
}
