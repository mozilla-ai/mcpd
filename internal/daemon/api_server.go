package daemon

import (
	"context"
	stdErrors "errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/hashicorp/go-hclog"

	"github.com/mozilla-ai/mcpd/v2/internal/api"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd/v2/internal/contracts"
	"github.com/mozilla-ai/mcpd/v2/internal/errors"
)

// APIServer manages the HTTP API for the daemon.
// NewAPIServer should be used to create instances of APIServer.
type APIServer struct {
	// Logger for API server operations.
	logger hclog.Logger

	// ClientManager handles MCP client connections.
	clientManager contracts.MCPClientAccessor

	// HealthTracker monitors server health status.
	healthTracker contracts.MCPHealthMonitor

	// Addr specifies the network address to bind.
	addr string

	// CORS configuration for cross-origin requests.
	cors CORSConfig

	// ShutdownTimeout specifies how long to wait for graceful shutdown.
	shutdownTimeout time.Duration
}

// NewAPIServer creates a new API server with the provided dependencies and options.
// Applies default options first, then user-provided options to ensure all fields have valid values.
func NewAPIServer(deps APIDependencies, opt ...APIOption) (*APIServer, error) {
	if err := deps.Validate(); err != nil {
		return nil, fmt.Errorf("invalid dependencies for API server: %w", err)
	}

	// Ensure we always start with defaults and apply user options on top.
	apiOpts, err := NewAPIOptions(opt...)
	if err != nil {
		return nil, fmt.Errorf("invalid API options: %w", err)
	}

	return &APIServer{
		logger:          deps.Logger.Named("api"),
		clientManager:   deps.ClientManager,
		healthTracker:   deps.HealthTracker,
		addr:            deps.Addr,
		cors:            apiOpts.CORS,
		shutdownTimeout: apiOpts.ShutdownTimeout,
	}, nil
}

// Start starts the API server and blocks until the context is canceled or an error occurs.
func (a *APIServer) Start(ctx context.Context) error {
	// Create router.
	mux := chi.NewMux()
	mux.Use(middleware.StripSlashes)

	// Add CORS middleware if enabled.
	if a.cors.Enabled {
		a.applyCORS(mux)
	}

	config := huma.DefaultConfig("mcpd docs", cmd.Version())
	router := humachi.New(mux, config)

	// Configure the error handling wrapping.
	huma.NewErrorWithContext = errorHandler(a.logger)

	// Safe way to ensure /api/v1.
	apiPathPrefix, err := url.JoinPath("/api", "v1")
	if err != nil {
		return err
	}

	// Group all routes under the /api/v1 prefix.
	v1 := huma.NewGroup(router, apiPathPrefix)
	api.RegisterHealthRoutes(v1, a.healthTracker, "/health")
	api.RegisterServerRoutes(v1, a.clientManager, "/servers")

	srv := &http.Server{
		Addr:    a.addr,
		Handler: mux,
	}
	errCh := make(chan error, 1)

	// Start the API.
	go func() {
		a.logger.Info("Starting API server", "address", a.addr, "prefix", apiPathPrefix)
		if a.cors.Enabled {
			a.logger.Info("CORS enabled", "origins", a.cors.AllowOrigins)
		}
		if err := srv.ListenAndServe(); err != nil && !stdErrors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	// Handle graceful shutdown.
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
		defer cancel()
		a.logger.Info("Shutting down API server...")
		_ = srv.Shutdown(shutdownCtx)
		a.logger.Info("Shutdown complete")
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

// applyCORS applies CORS middleware to the router based on the configured options.
func (a *APIServer) applyCORS(mux *chi.Mux) {
	a.logger.Info("Enabling CORS", "origins", a.cors.AllowOrigins)

	corsOptions := cors.Options{
		AllowedOrigins:   a.cors.AllowOrigins,
		AllowedMethods:   a.cors.AllowMethods,
		AllowedHeaders:   a.cors.AllowedHeaders,
		ExposedHeaders:   a.cors.ExposedHeaders,
		AllowCredentials: a.cors.AllowCredentials,
		MaxAge:           int(a.cors.MaxAge.Seconds()),
	}

	// Handle wildcard origins properly.
	for i, origin := range corsOptions.AllowedOrigins {
		if origin == "*" {
			corsOptions.AllowedOrigins = []string{"*"}
			corsOptions.AllowCredentials = false
			break
		}
		corsOptions.AllowedOrigins[i] = strings.TrimSpace(origin)
	}

	mux.Use(cors.Handler(corsOptions))
}

// mapError maps application domain errors to appropriate HTTP status codes.
//
// This function is the central place where domain errors from internal/errors are converted to HTTP responses.
// When adding new errors to internal/errors/errors.go, you MUST add them here to prevent them from falling
// through to the default case which returns HTTP 500.
//
// NOTE: Keep this function in sync with internal/errors/errors.go.
// Every error defined there should have an explicit case here otherwise it will default to 500.
//
// Mapping guidelines:
//   - 400: Client errors (bad input, invalid requests)
//   - 403: Authorization/permission errors
//   - 404: Resource not found errors
//   - 502: External service/dependency failures
//   - 500: Unexpected internal errors (default case)
//
// Don't forget to:
// 1. Add test cases to TestMapError (internal/daemon/api_server_test.go)
// 2. Update the documentation in internal/errors/errors.go
func mapError(logger hclog.Logger, err error) huma.StatusError {
	switch {
	case stdErrors.Is(err, errors.ErrBadRequest):
		return huma.Error400BadRequest(err.Error())
	case stdErrors.Is(err, errors.ErrServerNotFound):
		return huma.Error404NotFound(err.Error())
	case stdErrors.Is(err, errors.ErrToolsNotFound):
		return huma.Error404NotFound(err.Error())
	case stdErrors.Is(err, errors.ErrHealthNotTracked):
		return huma.Error404NotFound(err.Error())
	case stdErrors.Is(err, errors.ErrToolForbidden):
		return huma.Error403Forbidden(err.Error())
	case stdErrors.Is(err, errors.ErrToolListFailed):
		logger.Error("Tool list failed", "error", err)
		return huma.Error502BadGateway("MCP server error listing tools", err)
	case stdErrors.Is(err, errors.ErrToolCallFailed):
		logger.Error("Tool call failed", "error", err)
		return huma.Error502BadGateway("MCP server error calling tool", err)
	case stdErrors.Is(err, errors.ErrToolCallFailedUnknown):
		logger.Error("Tool call failed, unknown error", "error", err)
		return huma.Error502BadGateway("MCP server unknown error calling tool", err)
	default:
		logger.Error("Unexpected error interacting with MCP server", "error", err)
		return huma.Error500InternalServerError("Internal server error", err)
	}
}

// errorHandler wraps error handling for the application when converting to API friendly errors.
// It allows the logger to be supplied to functions that resolve huma.StatusError,
// and it supports different behaviors based on the variadic errors parameter.
func errorHandler(logger hclog.Logger) func(_ huma.Context, status int, msg string, errs ...error) huma.StatusError {
	return func(_ huma.Context, status int, msg string, errs ...error) huma.StatusError {
		switch len(errs) {
		case 0:
			// No errors provided; return a generic error.
			return huma.NewError(status, msg)
		case 1:
			// Single error; map it directly.
			return mapError(logger, errs[0])
		default:
			// Multiple errors; join them and map.
			combinedErr := stdErrors.Join(errs...)
			return mapError(logger, combinedErr)
		}
	}
}
