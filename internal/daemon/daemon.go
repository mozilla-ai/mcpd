package daemon

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/mozilla-ai/mcpd/v2/internal/config"
	configcontext "github.com/mozilla-ai/mcpd/v2/internal/context"
	"github.com/mozilla-ai/mcpd/v2/internal/flags"
	"github.com/mozilla-ai/mcpd/v2/internal/runtime"
)

type Daemon struct {
	apiServer         *ApiServer
	logger            hclog.Logger
	clientManager     *ClientManager
	supportedRuntimes map[runtime.Runtime]struct{}
	cfgLoader         config.Loader
}

func NewDaemon(logger hclog.Logger, cfgLoader config.Loader, apiAddr string) (*Daemon, error) {
	if logger == nil || reflect.ValueOf(logger).IsNil() {
		return nil, fmt.Errorf("logger cannot be nil")
	}
	if cfgLoader == nil || reflect.ValueOf(cfgLoader).IsNil() {
		return nil, fmt.Errorf("config loader cannot be nil")
	}
	if err := IsValidAddr(apiAddr); err != nil {
		return nil, fmt.Errorf("invalid api address '%s': %w", apiAddr, err)
	}

	clientManager := NewClientManager()

	apiServer, err := NewApiServer(logger, clientManager, apiAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create daemon API server: %w", err)
	}

	return &Daemon{
		logger:            logger.Named("daemon"),
		clientManager:     clientManager,
		apiServer:         apiServer,
		supportedRuntimes: runtime.DefaultSupportedRuntimes(),
		cfgLoader:         cfgLoader,
	}, nil
}

func (d *Daemon) LoadConfig() ([]runtime.Server, error) {
	cfgPath := flags.ConfigFile
	cfg, err := d.cfgLoader.Load(cfgPath)
	if err != nil {
		return nil, err
	}

	// Use the home directory to load the execution context config data (for now).
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not determine home directory: %w", err)
	}
	executionCtxPath := filepath.Join(home, ".mcpd", "secrets.dev.toml")
	execCtx, err := configcontext.LoadExecutionContextConfig(executionCtxPath)
	if err != nil {
		return nil, err
	}

	return runtime.AggregateConfigs(cfg, execCtx)
}

func (d *Daemon) StartAndManage(ctx context.Context) error {
	runtimeCfg, err := d.LoadConfig()
	if err != nil {
		return err
	}

	d.logger.Info(fmt.Sprintf("loaded config for %d daemon(s)", len(runtimeCfg)))
	fmt.Println(fmt.Sprintf("Attempting to start %d MCP server(s)", len(runtimeCfg)))

	var startupWg sync.WaitGroup
	d.setupSignalHandler()

	// Launch all MCP servers
	startupWg.Add(len(runtimeCfg))
	for _, r := range runtimeCfg {
		go func(server runtime.Server) {
			err := d.launchServer(ctx, server, &startupWg)
			if err != nil {
				d.logger.Error("failed to launch server", "error", err)
			}
		}(r)
	}

	startupWg.Wait()
	fmt.Println("MCP server started")

	// TODO: Configurable intervals/timeouts.
	healthcheckInterval := 10 * time.Second
	pingTimeout := 3 * time.Second

	go d.healthCheckLoop(ctx, healthcheckInterval, pingTimeout)

	readyChan := make(chan struct{})
	go func() {
		if err := d.apiServer.Start(readyChan); err != nil {
			d.logger.Error(fmt.Sprintf("API server failed: %s", err))
		}
	}()

	<-readyChan
	fmt.Println("Press CTRL+C to shut down.")
	select {}
}

func (d *Daemon) setupSignalHandler() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down all servers...")

		for _, name := range d.clientManager.List() {
			c, ok := d.clientManager.Client(name)
			if !ok {
				continue
			}
			d.logger.Info("Closing client connection to MCP server", "name", name)
			if err := c.Close(); err != nil {
				d.logger.Error("Error closing client connection to MCP server", "name", name, "error", err)
			}
		}
		os.Exit(0) // TODO: should we be exiting for everyone here?
	}()
}

func (d *Daemon) launchServer(ctx context.Context, server runtime.Server, wg *sync.WaitGroup) error {
	defer wg.Done()

	runtimeBinary := server.Runtime()
	_, supported := d.supportedRuntimes[runtime.Runtime(runtimeBinary)]
	if !supported {
		return fmt.Errorf("unsupported runtime/repository '%s' for MCP server daemon '%s'", runtimeBinary, server.Name)
	}

	// Strip arbitrary package prefix (e.g. uvx::)
	packageNameAndVersion := strings.TrimPrefix(server.Package, runtimeBinary+"::")
	env := server.Environ()
	var args []string
	// TODO: npx requires '-y' before the package name
	if runtime.Runtime(runtimeBinary) == runtime.NPX {
		args = append(args, "y")
	}
	args = append([]string{packageNameAndVersion}, server.Args...)

	d.logger.Info(
		"attempting to start MCP server",
		"name", server.Name,
		"binary", runtimeBinary,
		"args", args,
		"environment", env,
	)
	fmt.Println(fmt.Sprintf("Starting MCP server: '%s'...", server.Name))

	stdioClient, err := client.NewStdioMCPClient(runtimeBinary, env, args...)
	if err != nil {
		return fmt.Errorf("error starting MCP server: '%s': %v", server.Name, err)
	}

	d.logger.Info(fmt.Sprintf("MCP server started: '%s'...", server.Name))

	// Get stderr reader
	stderr, ok := client.GetStderr(stdioClient)
	if !ok {
		return fmt.Errorf("failed to get stderr from new MCP client: '%s'", server.Name)
	}

	// Pipe stderr to logger and terminal
	// TODO: Properly fix up the contexts and closing down of things.
	stdErrCtx, stdErrCancel := context.WithCancel(ctx)
	go func(ctx context.Context) {
		reader := bufio.NewReader(stderr)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				line, err := reader.ReadString('\n')
				if err != nil {
					if err != io.EOF {
						d.logger.Error("Error reading stderr", "error", err)
					}
					return
				}
				fmt.Println(line)
				d.logger.Info("stderr", "line", line)
			}
		}
	}(stdErrCtx)
	defer stdErrCancel()

	initializeCtx, cancel := context.WithTimeout(ctx, 30*time.Second) // TODO: Configurable timeout.
	defer cancel()

	// 'Initialize'
	initResult, err := stdioClient.Initialize(
		initializeCtx,
		mcp.InitializeRequest{
			Params: mcp.InitializeParams{
				ProtocolVersion: "latest",
				ClientInfo:      mcp.Implementation{Name: "mcpd", Version: "0.0.1"},
			},
		})
	if err != nil {
		return fmt.Errorf("error initializing MCP client: '%s': %w", server.Name, err)
	}

	packageNameAndVersion = fmt.Sprintf("%s@%s", initResult.ServerInfo.Name, initResult.ServerInfo.Version)
	d.logger.Info(fmt.Sprintf("Initialized MCP server: '%s': %s", server.Name, packageNameAndVersion))

	// Store the client.
	d.clientManager.Add(server.Name, stdioClient, server.Tools)

	d.logger.Info(fmt.Sprintf("Server '%s' ready.", server.Name))

	return nil
}

func (d *Daemon) healthCheckLoop(
	ctx context.Context,
	interval time.Duration,
	timeout time.Duration,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	d.pingAllServers(ctx, timeout)

	for {
		select {
		case <-ctx.Done():
			d.logger.Info("Stopping MCP server health checks")
			return
		case <-ticker.C:
			d.pingAllServers(ctx, timeout)
		}
	}
}

func (d *Daemon) pingAllServers(ctx context.Context, timeout time.Duration) {
	for _, name := range d.clientManager.List() {
		c, ok := d.clientManager.Client(name)
		if !ok {
			continue
		}

		go func(name string, mcpClient *client.Client) {
			pingCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			if err := mcpClient.Ping(pingCtx); err != nil {
				d.logger.Error(fmt.Sprintf("Error pinging MCP server: '%s'", name), "error", err)
				return
			}

			// TODO: Store health state for servers, and expose HTTP API route for /heath
			d.logger.Debug("Ping successful", "server", name)
		}(name, c)
	}
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
