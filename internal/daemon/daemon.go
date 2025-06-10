package daemon

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/hashicorp/go-hclog"

	"github.com/mozilla-ai/mcpd-cli/v2/internal/config"
	configcontext "github.com/mozilla-ai/mcpd-cli/v2/internal/context"
	"github.com/mozilla-ai/mcpd-cli/v2/internal/flags"
	"github.com/mozilla-ai/mcpd-cli/v2/internal/runtime"
)

type Daemon struct {
	apiServer                *ApiServer
	logger                   hclog.Logger
	clients                  map[string]*client.Client
	mu                       *sync.RWMutex
	repositoryBinaryMappings map[string]string
}

func NewDaemon(logger hclog.Logger) *Daemon {
	clients := make(map[string]*client.Client)
	clientsMutex := &sync.RWMutex{}
	l := logger.Named("daemon")
	return &Daemon{
		logger:  l,
		clients: clients,
		mu:      clientsMutex,
		repositoryBinaryMappings: map[string]string{
			"pypi": "uvx",
		},
		apiServer: &ApiServer{
			logger:       l,
			clients:      clients,
			clientsMutex: clientsMutex,
			serverTools:  make(map[string][]string),
		},
	}
}

func (d *Daemon) LoadConfig() ([]runtime.RuntimeServer, error) {
	cfgPath := flags.ConfigFile
	cfg, err := config.LoadConfig(cfgPath)
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

	// Allow the API server to track which tools are allowed for specific MCP servers.
	for _, r := range runtimeCfg {
		if len(r.Tools) > 0 {
			d.apiServer.serverTools[r.Name] = r.Tools
		}
	}

	d.logger.Info(fmt.Sprintf("loaded config for %d daemon(s)", len(runtimeCfg)))
	fmt.Println(fmt.Sprintf("Attempting to start %d MCP server(s)", len(runtimeCfg)))

	var startupWg sync.WaitGroup

	d.setupSignalHandler()

	// Launch all MCP servers
	startupWg.Add(len(runtimeCfg))
	for _, r := range runtimeCfg {
		go func() {
			err := d.launchServer(ctx, r, &startupWg)
			if err != nil {
				d.logger.Error("failed to launch server", "error", err)
			}
		}()
	}

	startupWg.Wait()
	fmt.Println("MCP server started")

	// TODO: Configurable?
	healthcheckInterval := 10 * time.Second
	pingTimeout := 3 * time.Second

	go d.healthCheckLoop(ctx, healthcheckInterval, pingTimeout)

	readyChan := make(chan struct{})

	go func() {
		err := d.apiServer.Start(8090, readyChan) // TODO: Pass in.
		if err != nil {
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
		d.mu.Lock()
		for name, c := range d.clients {
			log.Printf("Closing connection to '%s'...", name)
			// Use the library's Close method for graceful shutdown.
			if err := c.Close(); err != nil {
				log.Printf("Error closing client for '%s': %v", name, err)
			}
		}
		d.mu.Unlock()
		os.Exit(0)
	}()
}

func (d *Daemon) launchServer(ctx context.Context, server runtime.RuntimeServer, wg *sync.WaitGroup) error {
	defer wg.Done()

	currentRuntime := server.Runtime()
	binary, supported := d.repositoryBinaryMappings[currentRuntime]
	if !supported {
		return fmt.Errorf("unsupported runtime/repository '%s' for MCP server daemon '%s'", currentRuntime, server.Name)
	}

	packageName := strings.Split(strings.TrimPrefix(server.Package, currentRuntime+"::"), "@")[0]
	env := server.Environ()
	// args := append([]string{"--verbose", packageName}, server.Args...)
	args := append([]string{packageName}, server.Args...)

	d.logger.Info(
		"attempting to start MCP server",
		"name", server.Name,
		"binary", binary,
		"args", args,
		"environment", env,
	)
	fmt.Println(fmt.Sprintf("Starting MCP server: '%s'...", server.Name))

	stdioClient, err := client.NewStdioMCPClient(binary, env, args...)
	if err != nil {
		return fmt.Errorf("error starting MCP server: '%s': %v", server.Name, err)
	}

	// Get stderr reader
	stderr, ok := client.GetStderr(stdioClient)
	if !ok {
		log.Fatalf("Failed to get stderr from MCP client")
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

	initializeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 'Initialize'
	initResult, err := stdioClient.Initialize(
		initializeCtx,
		mcp.InitializeRequest{
			Params: mcp.InitializeParams{
				ProtocolVersion: "latest",
				// Capabilities:    mcp.ClientCapabilities{},
				ClientInfo: mcp.Implementation{Name: "mcpd", Version: "0.0.2"},
			},
		})
	if err != nil {
		return fmt.Errorf("error initializing MCP client: '%s': %w", server.Name, err)
	}

	nameAndVersion := fmt.Sprintf("%s@%s", initResult.ServerInfo.Name, initResult.ServerInfo.Version)
	d.logger.Info(fmt.Sprintf("Initialized MCP client: '%s': %s", server.Name, nameAndVersion))

	// Store the client.
	d.mu.Lock()
	d.clients[server.Name] = stdioClient
	d.mu.Unlock()

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
	d.mu.Lock()
	clientsCopy := make(map[string]*client.Client, len(d.clients))
	for k, v := range d.clients {
		clientsCopy[k] = v
	}
	d.mu.Unlock()

	for name, c := range clientsCopy {
		go func(name string, mcpClient *client.Client) {
			pingCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			if err := mcpClient.Ping(pingCtx); err != nil {
				d.logger.Error(fmt.Sprintf("Error pinging MCP server: '%s'", name), "error", err)
				return
			}

			d.logger.Info(fmt.Sprintf("Ping successful for MCP server: '%s'", name))
		}(name, c)
	}
}
