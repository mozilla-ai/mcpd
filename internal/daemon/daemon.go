package daemon

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"golang.org/x/sync/errgroup"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd/v2/internal/contracts"
	"github.com/mozilla-ai/mcpd/v2/internal/domain"
	"github.com/mozilla-ai/mcpd/v2/internal/filter"
	"github.com/mozilla-ai/mcpd/v2/internal/runtime"
)

// Daemon manages MCP server lifecycles, client connections, and health monitoring.
// It should only be created using NewDaemon to ensure proper initialization.
type Daemon struct {
	apiServer         *APIServer
	logger            hclog.Logger
	clientManager     contracts.MCPClientAccessor
	healthTracker     contracts.MCPHealthMonitor
	supportedRuntimes map[runtime.Runtime]struct{}
	runtimeServers    []runtime.Server

	// clientInitTimeout is the time allowed for MCP servers to initialize.
	clientInitTimeout time.Duration

	// clientShutdownTimeout is the time allowed for MCP servers to shut down.
	clientShutdownTimeout time.Duration

	// clientHealthCheckTimeout is the time allowed for an MCP server to respond to a health check (ping).
	clientHealthCheckTimeout time.Duration

	// clientHealthCheckInterval is the time interval between MCP server health checks (pings).
	clientHealthCheckInterval time.Duration
}

// NewDaemon creates a new Daemon instance with proper initialization.
// Use this function instead of directly creating a Daemon struct.
func NewDaemon(deps Dependencies, opt ...Option) (*Daemon, error) {
	if err := deps.Validate(); err != nil {
		return nil, fmt.Errorf("invalid daemon dependencies: %w", err)
	}

	// Ensure we always start with defaults and apply user options on top.
	opts, err := NewOptions(opt...)
	if err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	var serverNames []string // Track server names for server health tracker creation.
	var validateErrs error
	for _, srv := range deps.RuntimeServers {
		serverNames = append(serverNames, srv.Name())
		// Validate the config since the daemon will be required to start MCP servers using it.
		if err := srv.Validate(); err != nil {
			validateErrs = errors.Join(
				validateErrs,
				fmt.Errorf("invalid server configuration '%s': %w", srv.Name(), err),
			)
		}
	}
	if validateErrs != nil {
		// NOTE: Include a line break in the output to improve readability of the validation errors.
		return nil, errors.Join(fmt.Errorf("invalid runtime configuration"), validateErrs)
	}

	healthTracker := NewHealthTracker(serverNames)
	clientManager := NewClientManager()
	apiDeps, err := NewAPIDependencies(
		deps.Logger,
		clientManager,
		healthTracker,
		deps.APIAddr,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create API dependencies: %w", err)
	}

	apiServer, err := NewAPIServer(apiDeps, opts.APIOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create daemon API server: %w", err)
	}

	return &Daemon{
		logger:                    deps.Logger.Named("daemon"),
		clientManager:             clientManager,
		healthTracker:             healthTracker,
		apiServer:                 apiServer,
		supportedRuntimes:         runtime.DefaultSupportedRuntimes(),
		runtimeServers:            deps.RuntimeServers,
		clientInitTimeout:         opts.ClientInitTimeout,
		clientShutdownTimeout:     opts.ClientShutdownTimeout,
		clientHealthCheckTimeout:  opts.ClientHealthCheckTimeout,
		clientHealthCheckInterval: opts.ClientHealthCheckInterval,
	}, nil
}

// StartAndManage is a long-running method that starts configured MCP servers, and the API.
// It launches regular health checks on the MCP servers, with statuses visible via API routes.
func (d *Daemon) StartAndManage(ctx context.Context) error {
	// Handle clean-up.
	defer d.closeAllClients()

	// Launch servers
	if err := d.startMCPServers(ctx); err != nil {
		return err
	}

	// Run API and regular health checks.
	runGroup, runGroupCtx := errgroup.WithContext(ctx)
	runGroup.Go(func() error { return d.apiServer.Start(runGroupCtx) })
	runGroup.Go(func() error {
		return d.healthCheckLoop(runGroupCtx, d.clientHealthCheckInterval, d.clientHealthCheckTimeout)
	})

	return runGroup.Wait()
}

// startMCPServers launches every runtime server concurrently.
// It returns a combined error containing one entry per failed launch
// (nil if all servers start successfully).
func (d *Daemon) startMCPServers(ctx context.Context) error {
	errs := make([]error, 0, len(d.runtimeServers))
	mu := sync.Mutex{}
	runGroup, runCtx := errgroup.WithContext(ctx)

	for _, s := range d.runtimeServers {
		s := s
		runGroup.Go(func() error {
			if err := d.startMCPServer(runCtx, s); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("%s: %w", s.Name(), err))
				mu.Unlock()
			}
			// Since errors are collected, return nil to prevent context cancellation.
			return nil
		})
	}

	_ = runGroup.Wait()
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// startMCPServer starts a single MCP server and registers it with the daemon.
// It validates that the server has tools and a supported runtime before initializing.
func (d *Daemon) startMCPServer(ctx context.Context, server runtime.Server) error {
	// Validate that the server has tools configured.
	if len(server.Tools) == 0 {
		return fmt.Errorf(
			"server '%s' has no tools configured - MCP servers require at least one tool to function",
			server.Name(),
		)
	}

	runtimeBinary := server.Runtime()
	if _, supported := d.supportedRuntimes[runtime.Runtime(runtimeBinary)]; !supported {
		return fmt.Errorf(
			"unsupported runtime/repository '%s' for MCP server daemon '%s'",
			runtimeBinary,
			server.Name(),
		)
	}

	logger := d.logger.Named("mcp").Named(server.Name())
	logger.Info("Starting MCP server", "runtime", runtimeBinary, "package", server.Package)

	// Strip arbitrary package prefix (e.g. uvx::)
	packageNameAndVersion := strings.TrimPrefix(server.Package, runtimeBinary+"::")

	var args []string
	var environ []string

	// Handle runtime-specific setup
	switch runtime.Runtime(runtimeBinary) {
	case runtime.NPX:
		// NPX requires '-y' before the package name
		args = append(args, "-y")
		args = append(args, packageNameAndVersion)
		environ = server.SafeEnv()

	case runtime.Docker:
		// Docker requires special handling for stdio and environment variables
		// Note: Docker stderr (pull messages, startup logs) is redirected to prevent MCP protocol interference
		args = []string{"run", "-i", "--rm", "--network", "host"}

		// Pass environment variables as Docker -e flags
		for k, v := range server.Env {
			args = append(args, "-e", fmt.Sprintf("%s=%s", k, v))
		}

		// Add the image name
		args = append(args, packageNameAndVersion)

		// Docker doesn't need environ passed - env vars are handled via -e flags
		environ = nil

		// Override the binary name to "docker"
		runtimeBinary = "docker"

	default:
		// Default case (UVX and others)
		args = append(args, packageNameAndVersion)
		environ = server.SafeEnv()
	}

	args = append(args, server.Args...)

	logger.Debug("attempting to start server", "binary", runtimeBinary, "args", args)

	stdioClient, err := client.NewStdioMCPClient(runtimeBinary, environ, args...)
	if err != nil {
		return fmt.Errorf("error starting MCP server: '%s': %w", server.Name(), err)
	}

	logger.Info("Started")

	// Get stderr reader
	stderr, ok := client.GetStderr(stdioClient)
	if !ok {
		return fmt.Errorf("failed to get stderr from new MCP client: '%s'", server.Name())
	}

	// Pipe stderr to logger and terminal
	go func(ctx context.Context, logger hclog.Logger, stderr io.Reader) {
		reader := bufio.NewReader(stderr)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				line, err := reader.ReadString('\n')
				if ctx.Err() != nil {
					// Context canceled — probably shutting down, don't log the I/O error
					return
				}
				if err != nil && err != io.EOF {
					logger.Error("Error reading stderr", "error", err)
					return
				}

				parseAndLogMCPMessage(logger, line)
			}
		}
	}(ctx, logger, stderr)

	initializeCtx, cancel := context.WithTimeout(ctx, d.clientInitTimeout)
	defer cancel()

	// 'Initialize'
	initResult, err := stdioClient.Initialize(
		initializeCtx,
		mcp.InitializeRequest{
			Params: mcp.InitializeParams{
				ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
				ClientInfo:      mcp.Implementation{Name: cmd.AppName(), Version: cmd.Version()},
			},
		})
	if err != nil {
		return fmt.Errorf("error initializing MCP client: '%s': %w", server.Name(), err)
	}

	packageNameAndVersion = fmt.Sprintf("%s@%s", initResult.ServerInfo.Name, initResult.ServerInfo.Version)
	logger.Info(fmt.Sprintf("Initialized: '%s'", packageNameAndVersion))

	// Store and track the client.
	d.clientManager.Add(server.Name(), stdioClient, server.Tools)
	d.healthTracker.Add(server.Name())

	logger.Info("Ready!")

	return nil
}

// healthCheckLoop performs health checks (pings) on all servers.
// Will repeat at the specified interval until the supplied context is cancelled.
func (d *Daemon) healthCheckLoop(ctx context.Context, interval time.Duration, maxTimeout time.Duration) error {
	d.logger.Info("Starting health check loop", "interval", interval, "timeout", maxTimeout)

	// Bootstrap health monitoring before starting the loop.
	if err := d.pingAllServers(ctx, maxTimeout); err != nil {
		return err
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Start the loop that pings all servers each time the timer ticks,
	// continues until the context is cancelled.
	for {
		select {
		case <-ctx.Done():
			d.logger.Info("Stopping health check loop")
			return ctx.Err()
		case <-ticker.C:
			err := d.pingAllServers(ctx, maxTimeout)
			if err != nil {
				d.logger.Error("Error pinging all servers", "error", err)
			}
		}
	}
}

// pingServer attempts to ping a named registered MCP server and updates the MCPHealthMonitor with the result.
func (d *Daemon) pingServer(ctx context.Context, name string) error {
	// Early exit if context is already cancelled.
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	c, ok := d.clientManager.Client(name)
	if !ok {
		return fmt.Errorf("server '%s' not found", name)
	}

	start := time.Now()
	err := c.Ping(ctx)
	duration := time.Since(start)

	var status domain.HealthStatus
	var latency *time.Duration

	switch {
	case err == nil:
		status = domain.HealthStatusOK
		latency = &duration
		d.logger.Debug("Ping successful", "server", name, "latency", duration)
	case errors.Is(err, context.DeadlineExceeded):
		status = domain.HealthStatusTimeout
		d.logger.Error("Ping timed out", "server", name, "error", err)
	case errors.Is(err, context.Canceled):
		status = domain.HealthStatusTimeout
		d.logger.Warn("Ping context canceled", "server", name)
	default:
		status = domain.HealthStatusUnreachable
		d.logger.Error("Ping unreachable", "server", name, "error", err)
	}

	if updateErr := d.healthTracker.Update(name, status, latency); updateErr != nil {
		d.logger.Error("Failed to record health", "server", name, "error", updateErr)
		return updateErr
	}

	return nil
}

// pingAllServers attempts to ping all registered MCP servers and updates the MCPHealthMonitor with the results.
func (d *Daemon) pingAllServers(ctx context.Context, maxTimeout time.Duration) error {
	// Early exit if context is already cancelled.
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Ensure the maximum timeout is set (will be lower, if the context has less time left on it already).
	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, maxTimeout)
	defer timeoutCancel()

	g, gCtx := errgroup.WithContext(timeoutCtx)
	mu := sync.Mutex{}
	clients := d.clientManager.List()
	errs := make([]error, 0, len(clients))

	for _, name := range clients {
		name := name
		g.Go(func() error {
			if err := d.pingServer(gCtx, name); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
			// Since errors are collected, return nil to prevent context cancellation.
			return nil
		})
	}

	_ = g.Wait()
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// IsValidAddr returns an error if the address is not a valid "host:port" string.
func IsValidAddr(addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("invalid address format: %w", err)
	}

	if port == "" {
		return fmt.Errorf("address missing port")
	}

	// Try parsing port as a number
	if _, err := strconv.Atoi(port); err != nil {
		// Try looking up the named port
		if _, err := net.LookupPort("tcp", port); err != nil {
			return fmt.Errorf("invalid address port: %s", port)
		}
	}

	_ = host // it's ok to accept an empty host (listens on all interfaces)

	return nil
}

// parseAndLogMCPMessage parses a log line from the MCP server's stderr and logs it with the corresponding level.
func parseAndLogMCPMessage(logger hclog.Logger, line string) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return
	}

	// TODO: This format may change based on the runtime that spawned the MCP Server.
	// Attempt to parse the log format: LEVEL:LOGGER:MESSAGE.
	parts := strings.SplitN(trimmed, ":", 3)

	if len(parts) < 2 {
		logger.Info(trimmed)
		return
	}

	lvl := normalizeLogLevel(parts[0])
	message := parts[len(parts)-1]

	if lvl == hclog.NoLevel {
		logger.Info(trimmed)
		return
	}

	if lvl >= logger.GetLevel() {
		// The level is valid and at or above our logger's configured level.
		logger.Log(lvl, message)
	}

	// Either no logging (off) or a level we're not configured to log at.
}

func normalizeLogLevel(level string) hclog.Level {
	level = strings.TrimSpace(strings.ToLower(level))

	switch level {
	case "warning":
		return hclog.Warn // Normalize to warn
	case "fatal", "critical":
		return hclog.Error // Normalize to error
	default:
		return hclog.LevelFromString(level)
	}
}

// closeAllClients gracefully closes all managed clients with individual timeouts.
// It closes all clients concurrently and waits for all to complete or timeout.
func (d *Daemon) closeAllClients() {
	d.logger.Info("Shutting down MCP servers and client connections")

	clients := d.clientManager.List()
	if len(clients) == 0 {
		return
	}

	var wg sync.WaitGroup
	timeout := d.clientShutdownTimeout

	// Start closing all clients concurrently
	for _, n := range clients {
		name := n
		c, ok := d.clientManager.Client(name)
		if !ok {
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = d.closeClientWithTimeout(name, c, timeout) // Ignore return value - leaks are acceptable during shutdown
		}()
	}

	wg.Wait()
}

// closeClientWithTimeout closes a single client with a timeout.
// Returns true if the client closed successfully, false if it timed out.
func (d *Daemon) closeClientWithTimeout(name string, c client.MCPClient, timeout time.Duration) bool {
	d.logger.Info(fmt.Sprintf("Closing client %s", name))

	done := make(chan struct{})
	go func() {
		err := c.Close()
		if err != nil {
			// 'errors' can result in things like SIGINT which returns exit code 130,
			// we still log the error but only for debugging purposes.
			d.logger.Debug("Closing client", "client", name, "error", err)
		}
		close(done)
	}()

	// Wait for this specific client to close or timeout.
	select {
	case <-done:
		d.logger.Info(fmt.Sprintf("Closed client %s", name))
		return true
	case <-time.After(timeout):
		d.logger.Warn(
			fmt.Sprintf("Timeout (%s) closing client %s - process may still be running", timeout.String(), name),
		)
		return false
	}
}

// ReloadServers reloads the daemon's MCP servers based on a new configuration.
// It compares the current servers with the new configuration and:
// - Stops servers that have been removed
// - Starts servers that have been added
// - Updates tools for servers where only tools changed
// - Restarts servers with other configuration changes
// - Preserves servers that remain unchanged (keeping their client connections, tools, and health history intact)
func (d *Daemon) ReloadServers(ctx context.Context, newServers []runtime.Server) error {
	d.logger.Info("Starting server reload")

	// Validate all new servers before making any changes.
	var validateErrs error
	for _, srv := range newServers {
		if err := srv.Validate(); err != nil {
			srvErr := fmt.Errorf("invalid server configuration '%s': %w", srv.Name(), err)
			validateErrs = errors.Join(validateErrs, srvErr)
		}
	}
	if validateErrs != nil {
		return fmt.Errorf("server validation failed: %w", validateErrs)
	}

	existing := make(map[string]*runtime.Server)
	for _, srv := range d.runtimeServers {
		normalizedName := filter.NormalizeString(srv.Name())
		srvCopy := srv // Create a copy to get pointer
		srv.ServerEntry.Name = normalizedName
		existing[normalizedName] = &srvCopy
	}

	incoming := make(map[string]*runtime.Server, len(newServers))
	for _, srv := range newServers {
		normalizedName := filter.NormalizeString(srv.Name())
		srvCopy := srv // Create a copy to get pointer
		srv.ServerEntry.Name = normalizedName
		incoming[normalizedName] = &srvCopy
	}

	// Categorize changes.
	var toRemove []string
	var toAdd []*runtime.Server
	var toUpdateTools []*runtime.Server
	var toRestart []*runtime.Server
	var unchangedCount int

	// Find servers to remove (in current but not in new).
	for name := range existing {
		if _, exists := incoming[name]; !exists {
			toRemove = append(toRemove, name)
		}
	}

	// Find servers to add or modify (in new).
	for name, srv := range incoming {
		existingSrv, exists := existing[name]
		switch {
		case !exists:
			// New server
			toAdd = append(toAdd, srv)
		case existingSrv.Equals(srv):
			// No changes
			unchangedCount++
		case existingSrv.EqualsExceptTools(srv):
			// Only tools changed
			toUpdateTools = append(toUpdateTools, srv)
		default:
			// Other configuration changed - requires restart
			toRestart = append(toRestart, srv)
		}
	}

	d.logger.Info("Server configuration changes",
		"removed", len(toRemove),
		"added", len(toAdd),
		"tools_updated", len(toUpdateTools),
		"restarted", len(toRestart),
		"unchanged", unchangedCount)

	var errs []error

	// Stop removed servers.
	for _, name := range toRemove {
		if err := d.stopMCPServer(name); err != nil {
			d.logger.Error("Failed to stop server", "server", name, "error", err)
			errs = append(errs, fmt.Errorf("stop %s: %w", name, err))
		}
	}

	// Update tools for servers with tools-only changes.
	for _, srv := range toUpdateTools {
		if err := d.clientManager.UpdateTools(srv.Name(), srv.Tools); err != nil {
			d.logger.Error("Failed to update tools", "server", srv.Name(), "error", err)
			errs = append(errs, fmt.Errorf("update-tools %s: %w", srv.Name(), err))
		} else {
			d.logger.Info("Updated tools", "server", srv.Name(), "tools", srv.Tools)
		}
	}

	// Restart servers with configuration changes.
	for _, srv := range toRestart {
		d.logger.Info("Restarting server due to configuration changes", "server", srv.Name())

		// Stop the existing server.
		if err := d.stopMCPServer(srv.Name()); err != nil {
			d.logger.Error("Failed to stop server for restart", "server", srv.Name(), "error", err)
			errs = append(errs, fmt.Errorf("restart-stop %s: %w", srv.Name(), err))
			continue
		}

		// Start the server with new configuration.
		if err := d.startMCPServer(ctx, *srv); err != nil {
			d.logger.Error("Failed to start server after restart", "server", srv.Name(), "error", err)
			errs = append(errs, fmt.Errorf("restart-start %s: %w", srv.Name(), err))
		}
	}

	// Start new servers.
	for _, srv := range toAdd {
		if err := d.startMCPServer(ctx, *srv); err != nil {
			d.logger.Error("Failed to start new server", "server", srv.Name(), "error", err)
			errs = append(errs, fmt.Errorf("add %s: %w", srv.Name(), err))
		}
	}

	// Update stored runtime servers after reload (even if some operations failed).
	d.runtimeServers = newServers

	if len(errs) > 0 {
		d.logger.Error("Server reload completed with errors", "error_count", len(errs))
		return errors.Join(append([]error{fmt.Errorf("server reload had %d errors", len(errs))}, errs...)...)
	}

	d.logger.Info("Server reload completed successfully")
	return nil
}

// stopMCPServer gracefully stops a single MCP server and removes it from tracking.
func (d *Daemon) stopMCPServer(name string) error {
	d.logger.Info("Stopping MCP server", "server", name)

	c, ok := d.clientManager.Client(name)
	if !ok {
		return fmt.Errorf("server '%s' not found", name)
	}

	// Always remove from managers to maintain consistency.
	d.clientManager.Remove(name)
	d.healthTracker.Remove(name)

	// Close the client with timeout.
	if closed := d.closeClientWithTimeout(name, c, d.clientShutdownTimeout); !closed {
		d.logger.Error(
			"MCP server stop timed out - process may still be running and could be leaked",
			"server", name,
			"timeout", d.clientShutdownTimeout,
		)

		return fmt.Errorf(
			"server '%s' failed to stop within timeout %v - process may be leaked",
			name,
			d.clientShutdownTimeout,
		)
	}

	d.logger.Info("MCP server stopped successfully", "server", name)
	return nil
}
