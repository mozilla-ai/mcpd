package daemon

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hashicorp/go-hclog"
	"github.com/mark3labs/mcp-go/mcp"
)

const (
	apiPathPrefix = "/api/v1/"
)

type ApiServer struct {
	clientManager *ClientManager
	healthTracker *HealthTracker
	logger        hclog.Logger
	addr          string
}

func NewApiServer(logger hclog.Logger, clientManager *ClientManager, healthTracker *HealthTracker, addr string) (*ApiServer, error) {
	if err := IsValidAddr(addr); err != nil {
		return nil, err
	}

	return &ApiServer{
		logger:        logger.Named("api"),
		clientManager: clientManager,
		healthTracker: healthTracker,
		addr:          addr,
	}, nil
}

func (a *ApiServer) Start(ready chan<- struct{}) error {
	r := chi.NewRouter()
	r.Use(middleware.StripSlashes)

	r.Route("/api/v1", func(r chi.Router) {
		a.registerServerRoutes(r)
		a.registerHealthRoutes(r)
	})

	// Optionally add Swagger routes or /docs here later
	// r.Get("/docs/*", httpSwagger.WrapHandler)

	a.logger.Info("HTTP REST API listening", "address", a.addr, "prefix", "/api/v1")

	fqdn := fmt.Sprintf("http://%s%sservers", a.addr, apiPathPrefix) // TODO: HTTP/HTTPS
	fmt.Printf("HTTP REST API listening on: '%s'\n", fqdn)

	// Signal ready just before blocking for serving the API
	close(ready)
	return http.ListenAndServe(a.addr, r)
}

func (a *ApiServer) registerServerRoutes(r chi.Router) {
	r.Get("/servers", a.serversHandler)
	r.Get("/servers/{server}", a.handleServer)
	r.Post("/servers/{server}/{tool}", a.handleServerCallTool)
}

func (a *ApiServer) registerHealthRoutes(r chi.Router) {
	r.Route("/health", func(r chi.Router) {
		r.Route("/servers", func(r chi.Router) {
			r.Get("/", a.handleHealthServers)
			r.Get("/{server}", a.handleHealthServer)
		})
	})
}

func (a *ApiServer) callTool(ctx context.Context, server, tool string, args map[string]any) (any, error) {
	mcpClient, clientOk := a.clientManager.Client(server)
	if !clientOk {
		return nil, fmt.Errorf("%w: %s", ErrServerNotFound, server)
	}

	allowedTools, toolsOk := a.clientManager.Tools(server)
	if !toolsOk || len(allowedTools) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrToolsNotFound, server)
	}

	if !slices.Contains(allowedTools, tool) {
		return nil, fmt.Errorf("%w: %s/%s", ErrToolForbidden, server, tool)
	}

	result, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      tool,
			Arguments: args,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %s/%s: %w", ErrToolCallFailed, server, tool, err)
	} else if result == nil {
		return nil, fmt.Errorf("%w: %s/%s: result was nil", ErrToolCallFailedUnknown, server, tool)
	} else if result.IsError {
		return nil, fmt.Errorf("%w: %s/%s: %v", ErrToolCallFailedUnknown, server, tool, a.extractMessage(result.Content))
	}

	return a.extractMessage(result.Content), nil
}

// extractMessage searches the provided content and returns the text from the first mcp.TextContent item encountered.
// If the slice is nil, empty, or contains no text content, an empty string is returned.
func (a *ApiServer) extractMessage(content []mcp.Content) string {
	message := ""
	if content == nil || len(content) == 0 {
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
