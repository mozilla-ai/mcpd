package plugin

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-hclog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"

	pluginv1 "github.com/mozilla-ai/mcpd-plugins-sdk-go/pkg/plugins/v1/plugins"

	"github.com/mozilla-ai/mcpd/v2/internal/config"
)

const (
	// defaultPluginStartTimeout is the maximum time to wait for a plugin process to start.
	defaultPluginStartTimeout = 10 * time.Second

	// defaultPluginCallTimeout is the maximum time to wait for a plugin RPC call.
	defaultPluginCallTimeout = 5 * time.Second

	// pluginGracefulStopTimeout is the time allowed for graceful plugin shutdown.
	pluginGracefulStopTimeout = 5 * time.Second

	// pluginForceKillTimeout is the time to wait before force killing a plugin process.
	pluginForceKillTimeout = 2 * time.Second

	// socketPollInterval is how often to check if a socket is ready.
	socketPollInterval = 50 * time.Millisecond

	// socketDialTimeout is the timeout for individual socket dial attempts.
	socketDialTimeout = 100 * time.Millisecond

	// tcpPortRangeStart is the starting port for Windows TCP connections.
	tcpPortRangeStart = 50000

	// tcpPortRangeSize is the range of ports available for Windows TCP connections.
	tcpPortRangeSize = 10000

	// unixSocketIDRange is the range for unique socket file IDs.
	unixSocketIDRange = 1000000
)

const (
	networkTCP  = "tcp"
	networkUnix = "unix"
	osWindows   = "windows"
)

const (
	unixSchemePrefix = "unix://"
)

// Manager manages plugin processes and provides middleware for HTTP request/response processing.
// It starts plugins, maintains process control, and can force kill them at any time.
// Plugins are untrusted third party code.
// Use NewManager to create a Manager.
type Manager struct {
	logger       hclog.Logger
	config       *config.PluginConfig
	mu           sync.RWMutex
	plugins      map[string]*runningPlugin
	pipeline     *pipeline
	startTimeout time.Duration
	callTimeout  time.Duration

	// addressCounter is used to generate unique addresses for plugins.
	addressCounter atomic.Uint64
}

// runningPlugin tracks a plugin process and its gRPC connection.
type runningPlugin struct {
	logger   hclog.Logger
	cmd      *exec.Cmd
	conn     *grpc.ClientConn
	client   pluginv1.PluginClient
	instance *Instance
	address  string
	network  string
}

// NewManager creates a new plugin manager with the provided configuration.
func NewManager(logger hclog.Logger, cfg *config.PluginConfig) (*Manager, error) {
	if logger == nil || reflect.ValueOf(logger).IsNil() {
		return nil, fmt.Errorf("logger cannot be nil")
	}
	if cfg == nil {
		return nil, fmt.Errorf("plugin config cannot be nil")
	}

	// TODO: Extend Manager to accept options for timeouts.

	l := logger.Named("plugins")

	return &Manager{
		logger:       l,
		config:       cfg,
		plugins:      make(map[string]*runningPlugin),
		pipeline:     newPipeline(l),
		startTimeout: defaultPluginStartTimeout,
		callTimeout:  defaultPluginCallTimeout,
	}, nil
}

// StartPlugins discovers, starts, and registers all configured plugins.
// Returns an HTTP middleware function for request/response processing.
func (m *Manager) StartPlugins(ctx context.Context) (func(http.Handler) http.Handler, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Discover all executable binaries in plugins directory.
	pluginNames := m.config.PluginNamesDistinct()
	discovered, err := m.discoverPlugins(pluginNames)
	if err != nil {
		return nil, fmt.Errorf("error discovering plugins: %w", err)
	}
	if len(discovered) != len(pluginNames) {
		return nil, fmt.Errorf("missing configured plugins binaries")
	}

	m.logger.Info("discovered plugins", "count", len(discovered), "dir", m.config.Dir)

	// Start and register plugins for each category.
	for category, pluginEntries := range m.config.AllCategories() {
		for _, pluginEntry := range pluginEntries {
			// Find matching binary (this should always be fine).
			binaryPath, found := discovered[pluginEntry.Name]
			if !found {
				return nil, fmt.Errorf("plugin '%s' binary path not found", pluginEntry.Name)
			}

			// Start the plugin process.
			plg, err := m.startPlugin(ctx, pluginEntry.Name, binaryPath)
			if err != nil {
				return nil, fmt.Errorf("plugin '%s' failed to start: '%s': %w", pluginEntry.Name, binaryPath, err)
			}

			// Validate the plugin (check hashes match etc.).
			if err := plg.validate(ctx, pluginEntry); err != nil {
				return nil, errors.Join(
					fmt.Errorf("plugin '%s' validation error: %w", pluginEntry.Name, err),
					plg.stop(ctx),
				)
			}

			m.logger.Info("plugin started", "name", pluginEntry.Name, "pid", plg.cmd.Process.Pid)

			// Set required flag if configured.
			if pluginEntry.Required != nil {
				plg.instance.SetRequired(*pluginEntry.Required)
			}

			// Set the flows for which plugin execution should be allowed.
			plg.instance.SetFlows(pluginEntry.FlowsDistinct())

			// Track the plugin in the manager.
			m.plugins[pluginEntry.Name] = plg

			// Register with pipeline.
			if err := m.pipeline.Register(category, plg.instance); err != nil {
				return nil, fmt.Errorf("plugin '%s' registration error:: %w", pluginEntry.Name, err)
			}

			m.logger.Info("plugin registered",
				"name", pluginEntry.Name,
				"category", category,
				"required", plg.instance.Required(),
			)
		}
	}

	// Return middleware function.
	return m.pipeline.Middleware(), nil
}

// StopPlugins stops all running plugins.
// Force kills any that don't stop gracefully within the timeout.
func (m *Manager) StopPlugins(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error

	for name, plg := range m.plugins {
		if err := plg.stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("error stopping plugin '%s': %w", name, err))
		}
	}

	// Clear the plugins map (remove all).
	m.plugins = make(map[string]*runningPlugin)

	if len(errs) != 0 {
		return fmt.Errorf("errors stopping plugins: %w", errors.Join(errs...))
	}

	return nil
}

// validate attempts to get plugin metadata and use the plugin entry config to validate it.
// For example if the commit hash is configured then ensure it matches the reported commit hash.
func (p *runningPlugin) validate(ctx context.Context, pluginEntry config.PluginEntry) error {
	metadata, err := p.client.GetMetadata(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("failed to get metadata: %w", err)
	}

	// Return early if there's nothing to validate.
	if pluginEntry.CommitHash == nil || *pluginEntry.CommitHash == "" {
		return nil
	}

	// Return early if the commits match.
	if metadata.CommitHash == *pluginEntry.CommitHash {
		return nil
	}

	return fmt.Errorf("commit hash mismatch: expected %q, got %q", *pluginEntry.CommitHash, metadata.CommitHash)
}

// stop gracefully stops a single plugin.
// It attempts graceful shutdown first, then force kills if necessary.
func (p *runningPlugin) stop(ctx context.Context) error {
	if p == nil {
		return fmt.Errorf("plugin is nil")
	}

	var errs []error

	stopCtx, cancel := context.WithTimeout(ctx, pluginGracefulStopTimeout)
	defer cancel()
	if _, err := p.client.Stop(stopCtx, &emptypb.Empty{}); err != nil {
		errs = append(errs, fmt.Errorf("failed to stop plugin: %w", err))
	}

	if err := p.conn.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close plugin connection: %w", err))
	}

	// Try to close the process normally.
	done := make(chan error, 1)
	go func() {
		done <- p.cmd.Wait()
	}()

	select {
	case <-time.After(pluginForceKillTimeout):
		// Force kill the process if required.
		if err := p.cmd.Process.Kill(); err != nil {
			errs = append(errs, fmt.Errorf("failed to force kill plugin process: %w", err))
		}
		<-done
	case err := <-done:
		if err != nil {
			errs = append(errs, fmt.Errorf("plugin process exited with error: %w", err))
		}
	}

	// Clean up unix sockets.
	if p.network == networkUnix {
		if err := os.Remove(p.address); err != nil && !os.IsNotExist(err) {
			errs = append(errs, fmt.Errorf("failed to remove plugin's unix socket: %w", err))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// discoverPlugins scans the plugins directory for executable binaries that match the names of the configured plugins.
// Returns a map of plugin name to full binary path.
func (m *Manager) discoverPlugins(allowed map[string]struct{}) (map[string]string, error) {
	if len(allowed) == 0 {
		return nil, nil
	}

	entries, err := os.ReadDir(m.config.Dir)
	if err != nil {
		return nil, fmt.Errorf("reading plugin directory %s: %w", m.config.Dir, err)
	}

	plugins := make(map[string]string, len(allowed))

	for _, entry := range entries {
		// Skip directories.
		if entry.IsDir() {
			continue
		}

		// Skip hidden files.
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		// Skip any file that isn't in the configured allow list.
		if _, ok := allowed[entry.Name()]; !ok {
			continue
		}

		// Check if file is executable.
		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("reading file info for %s: %w", entry.Name(), err)
		}

		// Check for execute permission (0o111 = user/group/other execute bits).
		if info.Mode()&0o111 != 0 {
			fullPath := filepath.Join(m.config.Dir, entry.Name())
			plugins[entry.Name()] = fullPath
		}
	}

	return plugins, nil
}

// formatDialAddress formats the address for gRPC dialing based on network type.
func (m *Manager) formatDialAddress(network, address string) string {
	if network == networkUnix {
		return unixSchemePrefix + address
	}
	return address
}

// generateAddress generates a network address for a plugin based on the OS.
// Windows uses TCP, Unix systems use Unix domain sockets.
// Uses an atomic counter to ensure uniqueness across concurrent calls.
func (m *Manager) generateAddress(pluginName string) (address string, network string) {
	id := m.addressCounter.Add(1)

	switch runtime.GOOS {
	case osWindows:
		port := tcpPortRangeStart + (id % tcpPortRangeSize)
		return fmt.Sprintf("localhost:%d", port), networkTCP
	default:
		sockPath := filepath.Join(os.TempDir(), fmt.Sprintf("plugin-%s-%d.sock",
			strings.ReplaceAll(pluginName, " ", "-"),
			id%unixSocketIDRange))
		return sockPath, networkUnix
	}
}

// startPlugin launches a plugin binary, connects to it, and returns a Plugin instance.
// The manager maintains control of the process and can kill it at any time.
func (m *Manager) startPlugin(ctx context.Context, name string, binaryPath string) (*runningPlugin, error) {
	// Create logger per plugin.
	l := m.logger.Named(name)
	l.Info("starting plugin", "path", binaryPath)

	address, network := m.generateAddress(filepath.Base(binaryPath))
	l.Debug("transport selected", "network", network, "address", address)

	cmd := exec.CommandContext(ctx, binaryPath, "--address", address, "--network", network)

	// Use plugin specific logger to configure stdio and stderr for the plugin to emit logs.
	stdWriter := l.StandardWriter(&hclog.StandardLoggerOptions{
		InferLevels: true,
	})
	cmd.Stdout = stdWriter
	cmd.Stderr = stdWriter

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	l.Debug("plugin process started", "pid", cmd.Process.Pid, "address", address)

	dialCtx, cancel := context.WithTimeout(ctx, m.startTimeout)
	defer cancel()

	dialAddr := m.formatDialAddress(network, address)

	if err := m.waitForSocket(dialCtx, network, address); err != nil {
		if killErr := cmd.Process.Kill(); killErr != nil {
			l.Warn("failed to kill plugin process", "error", killErr)
		}
		return nil, fmt.Errorf("plugin didn't start in time: %w", err)
	}

	conn, err := grpc.NewClient(dialAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		if killErr := cmd.Process.Kill(); killErr != nil {
			l.Warn("failed to kill plugin process", "error", killErr)
		}
		return nil, fmt.Errorf("failed to connect to plugin: %w", err)
	}

	client := pluginv1.NewPluginClient(conn)

	adapter, err := NewGRPCAdapter(client, m.callTimeout)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = conn.Close()
		return nil, fmt.Errorf("error creating gRPC adapter: %w", err)
	}

	// Configure the plugin before checking its readiness.
	configCtx, configCancel := context.WithTimeout(ctx, m.callTimeout)
	defer configCancel()

	// TODO: Pass any supplied config (loaded from secrets.*.toml).

	if err := adapter.Configure(configCtx, nil); err != nil {
		return nil, fmt.Errorf("error configuring plugin: %w", err)
	}

	// Check if plugin is ready to handle requests before we return the plugin.
	readyCtx, readyCancel := context.WithTimeout(ctx, m.callTimeout)
	defer readyCancel()
	if err := adapter.CheckReady(readyCtx); err != nil {
		return nil, fmt.Errorf("plugin not ready: %w", err)
	}

	return &runningPlugin{
		logger: l,
		cmd:    cmd,
		conn:   conn,
		client: client,
		instance: &Instance{
			Plugin: adapter,
			name:   name,
		},
		address: address,
		network: network,
	}, nil
}

// waitForSocket polls the network address until it's ready or the context times out.
func (m *Manager) waitForSocket(ctx context.Context, network, address string) error {
	ticker := time.NewTicker(socketPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			conn, err := net.DialTimeout(network, address, socketDialTimeout)
			if err == nil {
				_ = conn.Close()
				return nil
			}
		}
	}
}
