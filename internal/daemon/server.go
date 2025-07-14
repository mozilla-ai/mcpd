package daemon

import (
	"context"
	stdErrors "errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hashicorp/go-hclog"

	"github.com/mozilla-ai/mcpd/v2/internal/api"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd/v2/internal/contracts"
	"github.com/mozilla-ai/mcpd/v2/internal/errors"
)

type ApiServer struct {
	clientManager contracts.MCPClientAccessor
	healthTracker contracts.MCPHealthMonitor
	logger        hclog.Logger
	addr          string
}

func NewApiServer(
	logger hclog.Logger,
	accessor contracts.MCPClientAccessor,
	monitor contracts.MCPHealthMonitor,
	addr string,
) (*ApiServer, error) {
	if logger == nil || reflect.ValueOf(logger).IsNil() {
		return nil, fmt.Errorf("logger cannot be nil")
	}
	if accessor == nil || reflect.ValueOf(accessor).IsNil() {
		return nil, fmt.Errorf("accessor cannot be nil")
	}
	if monitor == nil || reflect.ValueOf(monitor).IsNil() {
		return nil, fmt.Errorf("monitor cannot be nil")
	}
	if err := IsValidAddr(addr); err != nil {
		return nil, err
	}

	return &ApiServer{
		logger:        logger.Named("api"),
		clientManager: accessor,
		healthTracker: monitor,
		addr:          addr,
	}, nil
}

func (a *ApiServer) Start(ctx context.Context) error {
	// Create router.
	mux := chi.NewMux()
	mux.Use(middleware.StripSlashes)
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
		if err := srv.ListenAndServe(); err != nil && !stdErrors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	// Handle graceful shutdown.
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // TODO: make configurable
		defer cancel()
		a.logger.Info("Shutting down API server")
		_ = srv.Shutdown(shutdownCtx)
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

// mapError maps application domain errors to API errors.
func mapError(logger hclog.Logger, err error) huma.StatusError {
	switch {
	case stdErrors.Is(err, errors.ErrBadRequest):
		return huma.Error400BadRequest(err.Error())
	case stdErrors.Is(err, errors.ErrServerNotFound):
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
