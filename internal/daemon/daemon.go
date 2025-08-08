package daemon

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"golang.org/x/sync/errgroup"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
	configcontext "github.com/mozilla-ai/mcpd/v2/internal/context"
	"github.com/mozilla-ai/mcpd/v2/internal/contracts"
	"github.com/mozilla-ai/mcpd/v2/internal/domain"
	"github.com/mozilla-ai/mcpd/v2/internal/flags"
	"github.com/mozilla-ai/mcpd/v2/internal/runtime"
)

// Daemon manages MCP server lifecycles, client connections, and health monitoring.
// It should only be created using NewDaemon to ensure proper initialization.
type Daemon struct {
	apiServer         *ApiServer
	logger            hclog.Logger
	clientManager     contracts.MCPClientAccessor
	healthTracker     contracts.MCPHealthMonitor
	supportedRuntimes map[runtime.Runtime]struct{}
	runtimeCfg        []runtime.Server
}

type Opts struct {
	logger    hclog.Logger
	cfgLoader config.Loader
	ctxLoader configcontext.Loader
}

func NewDaemonOpts(logger hclog.Logger, cfgLoader config.Loader, ctxLoader configcontext.Loader) (*Opts, error) {
	if logger == nil || reflect.ValueOf(logger).IsNil() {
		return nil, fmt.Errorf("logger cannot be nil")
	}
	if cfgLoader == nil || reflect.ValueOf(cfgLoader).IsNil() {
		return nil, fmt.Errorf("config loader cannot be nil")
	}
	if ctxLoader == nil || reflect.ValueOf(ctxLoader).IsNil() {
		return nil, fmt.Errorf("runtime execution context config loader cannot be nil")
	}

	return &Opts{
		logger:    logger,
		cfgLoader: cfgLoader,
		ctxLoader: ctxLoader,
	}, nil
}

// NewDaemon creates a new Daemon instance with proper initialization.
// Use this function instead of directly creating a Daemon struct.
func NewDaemon(apiAddr string, opts *Opts) (*Daemon, error) {
	if err := IsValidAddr(apiAddr); err != nil {
		return nil, fmt.Errorf("invalid API address '%s': %w", apiAddr, err)
	}

	// Load config.
	cfg, err := loadConfig(opts.cfgLoader, opts.ctxLoader)
	if err != nil {
		return nil, err
	}

	var serverNames []string // Track server names for server health tracker creation.
	var validateErrs error
	for _, srv := range cfg {
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
		return nil, fmt.Errorf("invalid runtime configuration:\n%w", validateErrs)
	}

	healthTracker := NewHealthTracker(serverNames)
	clientManager := NewClientManager()
	apiServer, err := NewApiServer(opts.logger, clientManager, healthTracker, apiAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create daemon API server: %w", err)
	}

	return &Daemon{
		logger:            opts.logger.Named("daemon"),
		clientManager:     clientManager,
		healthTracker:     healthTracker,
		apiServer:         apiServer,
		supportedRuntimes: runtime.DefaultSupportedRuntimes(),
		runtimeCfg:        cfg,
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
	runGroup.Go(func() error { return d.healthCheckLoop(runGroupCtx, 10*time.Second, 3*time.Second) })

	return runGroup.Wait()
}

// startMCPServers launches every runtime server concurrently.
// It returns a combined error containing one entry per failed launch
// (nil if all servers start successfully).
func (d *Daemon) startMCPServers(ctx context.Context) error {
	errs := make([]error, 0, len(d.runtimeCfg))
	mu := sync.Mutex{}
	runGroup, runCtx := errgroup.WithContext(ctx)

	for _, s := range d.runtimeCfg {
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

func (d *Daemon) startMCPServer(ctx context.Context, server runtime.Server) error {
	runtimeBinary := server.Runtime()
	if _, supported := d.supportedRuntimes[runtime.Runtime(runtimeBinary)]; !supported {
		return fmt.Errorf(
			"unsupported runtime/repository '%s' for MCP server daemon '%s'",
			runtimeBinary,
			server.Name(),
		)
	}

	logger := d.logger.Named("mcp").Named(server.Name())

	// Strip arbitrary package prefix (e.g. uvx::)
	packageNameAndVersion := strings.TrimPrefix(server.Package, runtimeBinary+"::")

	var args []string
	// TODO: npx requires '-y' before the package name
	if runtime.Runtime(runtimeBinary) == runtime.NPX {
		args = append(args, "-y")
	}
	args = append(args, packageNameAndVersion)
	args = append(args, server.ResolvedArgs()...)

	logger.Debug("attempting to start server", "binary", runtimeBinary)

	stdioClient, err := client.NewStdioMCPClient(runtimeBinary, server.Environ(), args...)
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

	initializeCtx, cancel := context.WithTimeout(ctx, 30*time.Second) // TODO: Configurable timeout.
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

	// Store the client.
	d.clientManager.Add(server.Name(), stdioClient, server.Tools)

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

// pingServers attempts to ping all registered MCP server and updates the MCPHealthMonitor with the results.
func (d *Daemon) pingAllServers(ctx context.Context, maxTimeout time.Duration) error {
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

func loadConfig(cfgLoader config.Loader, ctxLoader configcontext.Loader) ([]runtime.Server, error) {
	cfg, err := cfgLoader.Load(flags.ConfigFile)
	if err != nil {
		return nil, err
	}

	execCtx, err := ctxLoader.Load(flags.RuntimeFile)
	if err != nil {
		return nil, err
	}

	return runtime.AggregateConfigs(cfg, execCtx)
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
	timeout := 5 * time.Second

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
			d.closeClientWithTimeout(name, c, timeout)
		}()
	}

	wg.Wait()
}

// closeClientWithTimeout closes a single client with a timeout.
func (d *Daemon) closeClientWithTimeout(name string, c client.MCPClient, timeout time.Duration) {
	d.logger.Info(fmt.Sprintf("Closing client %s", name))

	done := make(chan struct{})
	go func() {
		err := c.Close()
		if err != nil {
			// 'errors' can result in things like SIGINT which returns exit code 130,
			// we still log the error but only for debugging purposes.
			d.logger.Debug("Closing client", "client", name, "error", err)
		}
		d.logger.Info(fmt.Sprintf("Closed client %s", name))
		close(done)
	}()

	// Wait for this specific client to close or timeout.
	// NOTE: this could leak if we just time out clients,
	// but since we're exiting mcpd it isn't an issue.
	select {
	case <-done:
		// Closed successfully.
	case <-time.After(timeout):
		d.logger.Warn(fmt.Sprintf("Timeout (%s) closing client %s", timeout.String(), name))
	}
}
