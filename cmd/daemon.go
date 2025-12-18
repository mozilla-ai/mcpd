package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
	configcontext "github.com/mozilla-ai/mcpd/v2/internal/context"
	"github.com/mozilla-ai/mcpd/v2/internal/daemon"
	"github.com/mozilla-ai/mcpd/v2/internal/flags"
	"github.com/mozilla-ai/mcpd/v2/internal/runtime"
)

// Flag name constants for daemon command line flags.
const (
	// flagAddr is the flag name for the API server address.
	flagAddr = "addr"

	// flagCORSEnable is the flag name for enabling CORS support.
	flagCORSEnable = "cors-enable"

	// flagCORSCredentials is the flag name for CORS credentials support.
	flagCORSCredentials = "cors-allow-credentials"

	// flagCORSHeaders is the flag name for specifying allowed headers in CORS requests.
	flagCORSHeaders = "cors-allow-header"

	// flagCORSExposeHeaders is the flag name for specifying headers that are allowed
	// to be read by the client from CORS responses.
	flagCORSExposeHeaders = "cors-expose-header"

	// flagCORSOrigin is the flag name for allowed CORS origins.
	flagCORSOrigin = "cors-allow-origin"

	// flagCORSMethod is the flag name for allowed CORS methods.
	flagCORSMethod = "cors-allow-method"

	// flagCORSMaxAge is the flag name for CORS preflight cache duration.
	flagCORSMaxAge = "cors-max-age"

	// flagTimeoutAPIShutdown is the flag name for API server shutdown timeout.
	flagTimeoutAPIShutdown = "timeout-api-shutdown"

	// flagTimeoutMCPInit is the flag name for MCP server initialization timeout.
	flagTimeoutMCPInit = "timeout-mcp-init"

	// flagTimeoutMCPHealth is the flag name for MCP server health check timeout.
	flagTimeoutMCPHealth = "timeout-mcp-health"

	// flagTimeoutMCPShutdown is the flag name for timeout when shutting down the MCP client.
	flagTimeoutMCPShutdown = "timeout-mcp-shutdown"

	// flagIntervalMCPHealth is the flag name for MCP server health check interval.
	flagIntervalMCPHealth = "interval-mcp-health"
)

// DaemonCmd represents the 'daemon' command.
type DaemonCmd struct {
	*cmd.BaseCmd

	// dev indicates if daemon should run in development mode (not a config setting).
	dev bool

	// config contains all daemon configuration flags.
	config daemonFlagConfig

	// cfgLoader loads .mcpd.toml configuration.
	cfgLoader config.Loader

	// ctxLoader loads runtime context (secrets, env vars, args).
	ctxLoader configcontext.Loader
}

// daemonFlagConfig groups all daemon configuration flags.
type daemonFlagConfig struct {
	// api contains API server configuration.
	api apiFlagConfig

	// cors contains Cross-Origin Resource Sharing configuration.
	cors corsFlagConfig

	// timeout contains timeout-related configuration.
	timeout timeoutFlagConfig

	// interval contains interval-related configuration.
	interval intervalFlagConfig
}

// apiFlagConfig groups API server configuration flags.
type apiFlagConfig struct {
	// addr specifies the address to bind the daemon API server.
	addr string
}

// corsFlagConfig groups CORS-related configuration flags.
type corsFlagConfig struct {
	// enable determines if CORS should be enabled.
	enable bool

	// credentials determines if credentials are allowed in CORS requests.
	credentials bool

	// exposedHeaders determines the additional CORS response headers allowed to be read by the client.
	exposedHeaders []string

	// headers determines the additional request headers allowed to be sent in CORS requests.
	headers []string

	// methods specifies allowed HTTP methods for CORS requests.
	methods []string

	// origins specifies allowed origins for CORS requests.
	origins []string

	// maxAge specifies how long browsers can cache CORS preflight responses.
	maxAge string // duration string like "5m".
}

// timeoutFlagConfig groups timeout-related configuration flags.
type timeoutFlagConfig struct {
	// apiShutdown specifies how long to wait for graceful API server shutdown.
	apiShutdown string

	// mcpInit specifies how long to wait for MCP server initialization.
	mcpInit string

	// mcpShutdown specifies the maximum time to wait when attempting to shut down an MCP server.
	mcpShutdown string

	// healthCheck specifies how long to wait for health check responses from MCP servers.
	healthCheck string

	// clientShutdown specifies how long to wait for MCP clients to close (maps to ClientShutdownTimeout in daemon options).
	clientShutdown string
}

// intervalFlagConfig groups interval-related configuration flags.
type intervalFlagConfig struct {
	// healthCheck specifies how often to check MCP server's health.
	healthCheck string
}

// reloadState tracks the daemon's configuration reload state.
//
// The reloading flag is managed in two places:
//   - Set to true by handleSignals when SIGHUP is received
//   - Reset to false by the main reload loop after reload completes
//
// This ensures only one reload can be in progress at a time.
// Additional SIGHUP signals received while reloading is true are dropped with a warning.
//
// The reloadChan is a buffered channel (size 1) that queues reload requests.
// When handleSignals successfully sets the reloading flag, it sends to this channel.
// The main loop receives from this channel and performs the actual reload operation.
//
// Both fields are unexported as this type is only used internally within the daemon command implementation.
//
// Use newReloadState to construct a properly initialized reloadState with its cleanup function.
type reloadState struct {
	reloading  atomic.Bool
	reloadChan chan struct{}
}

// newReloadState creates a new reloadState with a buffered channel.
// Returns the state and a cleanup function that should be deferred to properly close the reload channel.
//
// Example:
//
//	state, cleanup := newReloadState()
//	defer cleanup()
func newReloadState() (state *reloadState, cancelFunc func()) {
	state = &reloadState{
		reloadChan: make(chan struct{}, 1),
	}
	cancelFunc = func() {
		close(state.reloadChan)
	}

	// Named returns.
	return
}

func newDaemonCmd(baseCmd *cmd.BaseCmd, cfgLoader config.Loader, ctxLoader configcontext.Loader) (*DaemonCmd, error) {
	if cfgLoader == nil || reflect.ValueOf(cfgLoader).IsNil() {
		return nil, fmt.Errorf("config loader cannot be nil")
	}

	if ctxLoader == nil || reflect.ValueOf(ctxLoader).IsNil() {
		return nil, fmt.Errorf("context loader cannot be nil")
	}

	c := &DaemonCmd{
		BaseCmd:   baseCmd,
		cfgLoader: cfgLoader,
		ctxLoader: ctxLoader,
	}

	return c, nil
}

func newDaemonCobraCmd(daemonCmd *DaemonCmd) *cobra.Command {
	cobraCommand := &cobra.Command{
		Use:   "daemon [--dev] [--addr] [--cors-enable] [--cors-origin]...",
		Short: "Launches an `mcpd` daemon instance",
		Long:  "Launches an `mcpd` daemon instance, which starts MCP servers and provides routing via HTTP API",
		RunE:  daemonCmd.run,
	}

	cobraCommand.Flags().BoolVar(
		&daemonCmd.dev,
		"dev",
		false,
		"Run the daemon in development-focused mode",
	)

	cobraCommand.Flags().StringVar(
		&daemonCmd.config.api.addr,
		flagAddr,
		"0.0.0.0:8090",
		"Address for the daemon to bind (not applicable in --dev mode)",
	)

	// Add CORS flags.
	cobraCommand.Flags().BoolVar(
		&daemonCmd.config.cors.enable,
		flagCORSEnable,
		false,
		"Enable Cross-Origin Resource Sharing (CORS) for browser clients.",
	)

	cobraCommand.Flags().StringSliceVar(
		&daemonCmd.config.cors.origins,
		flagCORSOrigin,
		nil,
		"Allowed CORS origin (can be repeated)",
	)

	cobraCommand.Flags().StringSliceVar(
		&daemonCmd.config.cors.methods,
		flagCORSMethod,
		daemon.DefaultCORSAllowMethods(),
		"Allowed CORS request method, e.g. 'GET' (can be repeated)",
	)

	cobraCommand.Flags().StringSliceVar(
		&daemonCmd.config.cors.headers,
		flagCORSHeaders,
		daemon.DefaultCORSAllowHeaders(),
		"Allowed CORS request header (can be repeated)",
	)

	cobraCommand.Flags().StringSliceVar(
		&daemonCmd.config.cors.exposedHeaders,
		flagCORSExposeHeaders,
		nil,
		"CORS response headers that should be made available to scripts in the browser (can be repeated)",
	)

	cobraCommand.Flags().BoolVar(
		&daemonCmd.config.cors.credentials,
		flagCORSCredentials,
		daemon.DefaultCORSAllowCredentials(),
		"Allow credentials in CORS requests",
	)

	cobraCommand.Flags().StringVar(
		&daemonCmd.config.cors.maxAge,
		flagCORSMaxAge,
		daemon.DefaultCORSMaxAge().String(),
		"CORS preflight max age (e.g., '5m', '300s')",
	)

	// Add timeout flags.
	cobraCommand.Flags().StringVar(
		&daemonCmd.config.timeout.apiShutdown,
		flagTimeoutAPIShutdown,
		daemon.DefaultAPIShutdownTimeout().String(),
		"Timeout in seconds to wait for graceful API server shutdown (e.g. 5s, 10s)",
	)

	cobraCommand.Flags().StringVar(
		&daemonCmd.config.timeout.mcpInit,
		flagTimeoutMCPInit,
		daemon.DefaultClientInitTimeout().String(),
		"Timeout in seconds to wait per MCP server for initialization (e.g. 5s, 10s)",
	)

	cobraCommand.Flags().StringVar(
		&daemonCmd.config.timeout.healthCheck,
		flagTimeoutMCPHealth,
		daemon.DefaultHealthCheckTimeout().String(),
		"Timeout in seconds to wait for completion of MCP server health checks (e.g. 5s, 10s)",
	)

	cobraCommand.Flags().StringVar(
		&daemonCmd.config.timeout.mcpShutdown,
		flagTimeoutMCPShutdown,
		daemon.DefaultClientShutdownTimeout().String(),
		"Timeout in seconds to wait for shutdown of MCP servers (e.g. 5s, 10s)",
	)

	// Add interval flags (aligned with daemon defaults).
	cobraCommand.Flags().StringVar(
		&daemonCmd.config.interval.healthCheck,
		flagIntervalMCPHealth,
		daemon.DefaultHealthCheckInterval().String(),
		"Time interval in seconds to wait between MCP server health check attempts (e.g. 5s, 10s)",
	)

	cobraCommand.MarkFlagsMutuallyExclusive("dev", flagAddr)

	// NOTE: Additional CORS validation required to check CORS flags are present alongside --cors-enable.
	cobraCommand.MarkFlagsRequiredTogether(flagCORSEnable, flagCORSOrigin)

	return cobraCommand
}

// NewDaemonCmd creates a newly configured (Cobra) command.
func NewDaemonCmd(baseCmd *cmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	opts, err := cmdopts.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	// Daemon requires plugin binary validation at load time.
	validatingLoader, err := config.NewValidatingLoader(
		opts.ConfigLoader,
		config.ValidatePluginBinaries,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create validating loader: %w", err)
	}

	daemonCmd, err := newDaemonCmd(baseCmd, validatingLoader, opts.ContextLoader)
	if err != nil {
		return nil, err
	}

	return newDaemonCobraCmd(daemonCmd), nil
}

// run is configured (via NewDaemonCmd) to be called by the Cobra framework when the command is executed.
// It may return an error (or nil, when there is no error).
func (c *DaemonCmd) run(cmd *cobra.Command, _ []string) error {
	logger, err := c.Logger()
	if err != nil {
		return err
	}

	if err := c.validateFlags(cmd); err != nil {
		return err
	}

	// Load the new configuration.
	cfg, err := c.LoadConfig(c.cfgLoader)
	if err != nil {
		return fmt.Errorf("%w: %w", config.ErrConfigLoadFailed, err)
	}

	// Load configuration layers (config file, then flag overrides).
	warnings, err := c.loadConfigurationLayers(logger, cmd, cfg)
	if err != nil {
		return err
	}

	if c.dev && len(warnings) > 0 {
		for _, warning := range warnings {
			fmt.Printf("Flag override: %s\n", warning)
		}
	}

	addr := strings.TrimSpace(c.config.api.addr)

	// Override address for dev mode.
	if c.dev {
		devAddr := "localhost:8090"
		fmt.Printf("--dev mode forces address: %s → %s\n", addr, devAddr)
		logger.Info("Development-focused mode", "addr", addr, "override", devAddr)
		addr = devAddr
	}

	// Validate the final address that will be used.
	if err := daemon.IsValidAddr(addr); err != nil {
		return err
	}

	execCtx, err := c.ctxLoader.Load(flags.RuntimeFile)
	if err != nil {
		return fmt.Errorf("failed to load runtime context: %w", err)
	}

	runtimeServers, err := runtime.AggregateConfigs(cfg, execCtx)
	if err != nil {
		return fmt.Errorf("failed to aggregate configs: %w", err)
	}

	deps, err := daemon.NewDependencies(logger, addr, runtimeServers)
	if err != nil {
		return fmt.Errorf("error creating daemon dependencies: %w", err)
	}

	apiOptions, err := c.buildAPIOptions()
	if err != nil {
		return fmt.Errorf("error creating API options: %w", err)
	}

	opts, err := c.buildDaemonOptions(apiOptions)
	if err != nil {
		return fmt.Errorf("error creating daemon options: %w", err)
	}

	// Add plugin configuration if present.
	if cfg.Plugins != nil {
		opts = append(opts, daemon.WithPluginConfig(cfg.Plugins))
	}

	d, err := daemon.NewDaemon(deps, opts...)
	if err != nil {
		return fmt.Errorf("failed to create mcpd daemon instance: %w", err)
	}

	// Create signal contexts for shutdown and reload.
	shutdownCtx, shutdownCancel := context.WithCancel(context.Background())
	defer shutdownCancel()

	// Setup signal handling.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	defer signal.Stop(sigChan)

	// Create reload channel for SIGHUP handling.
	state, cancelState := newReloadState()
	defer cancelState()

	runErr := make(chan error, 1)
	go func() {
		if err := d.StartAndManage(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
			runErr <- err
		}
		close(runErr)
	}()

	if c.dev {
		c.printDevBanner(cmd.OutOrStdout(), logger, addr)
	}

	// Start signal handling in background.
	go c.handleSignals(logger, sigChan, state, shutdownCancel)

	// Start the daemon's main loop which responds to reloads, shutdowns and startup errors.
	for {
		select {
		case <-state.reloadChan:
			if err := c.reloadServers(shutdownCtx, d); err != nil {
				logger.Error(
					"Failed to reload servers, exiting to prevent inconsistent state",
					"error", err,
				)
				return fmt.Errorf("configuration reload failed: %w", err)
			}

			// Mark reloading as complete.
			state.reloading.Store(false)
		case <-shutdownCtx.Done():
			logger.Info("Shutting down daemon...")
			err := <-runErr // Wait for cleanup and deferred logging
			logger.Info("Shutdown complete")

			return err // Graceful shutdown
		case err := <-runErr:
			if err != nil {
				logger.Error("daemon exited with error", "error", err)
				return err // Propagate daemon failure
			}

			return nil
		}
	}
}

// flagOverrideWarning creates a warning message for a flag overriding a config file value.
// It handles different types with appropriate formatting: []string gets comma-separated,
// everything else uses default %v formatting.
func flagOverrideWarning[T any](flagName string, configValue T, flagValue T) string {
	return fmt.Sprintf(
		"--%s: config=%s, flag=%s (using flag)",
		flagName,
		formatValue(configValue),
		formatValue(flagValue),
	)
}

// formatValue formats a value for user-friendly display in warnings.
func formatValue(value any) string {
	switch v := value.(type) {
	case []string:
		if len(v) == 0 {
			return "[]"
		}
		return "[" + strings.Join(v, ", ") + "]"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// loadConfigurationLayers loads config file values and applies flag overrides.
// It follows the precedence order: flags > config file > defaults.
// CLI flags override config file values when explicitly set.
// Returns warnings for each flag override and any error encountered.
func (c *DaemonCmd) loadConfigurationLayers(
	logger hclog.Logger,
	cmd *cobra.Command,
	cfg *config.Config,
) ([]string, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config data not present, cannot apply configuration layers")
	}

	// No daemon config section - flags and defaults will be used.
	if cfg.Daemon == nil {
		return nil, nil
	}

	var warnings []string
	daemonCfg := cfg.Daemon

	// Load config values, with flag overrides applied where explicitly set.
	warnings = append(warnings, c.loadConfigAPI(daemonCfg.API, logger, cmd)...)
	warnings = append(warnings, c.loadConfigMCP(daemonCfg.MCP, logger, cmd)...)

	return warnings, nil
}

// loadConfigAPI loads API configuration from config file, with flag overrides.
func (c *DaemonCmd) loadConfigAPI(api *config.APIConfigSection, logger hclog.Logger, cmd *cobra.Command) []string {
	if api == nil {
		return nil
	}

	var warnings []string

	// Handle API address.
	if api.Addr != nil {
		if cmd.Flags().Changed(flagAddr) {
			warnings = append(warnings, flagOverrideWarning(flagAddr, *api.Addr, c.config.api.addr))
			logger.Debug(
				"Flag overriding config value",
				"flag", flagAddr,
				"config", *api.Addr,
				"using", c.config.api.addr)
		} else {
			logger.Debug("Using config file value", "setting", "api.addr", "value", *api.Addr)
			c.config.api.addr = *api.Addr
		}
	}

	// Handle API timeout settings.
	if api.Timeout != nil {
		warnings = append(warnings, c.loadConfigAPITimeout(api.Timeout, logger, cmd)...)
	}

	// Handle CORS settings.
	if api.CORS != nil {
		warnings = append(warnings, c.loadConfigCORS(api.CORS, logger, cmd)...)
	}

	return warnings
}

// loadConfigAPITimeout loads API timeout configuration from config file, with flag overrides.
func (c *DaemonCmd) loadConfigAPITimeout(
	timeout *config.APITimeoutConfigSection,
	logger hclog.Logger,
	cmd *cobra.Command,
) []string {
	if timeout == nil {
		return nil
	}

	var warnings []string

	// Handle API shutdown timeout.
	if timeout.Shutdown != nil {
		parsed := timeout.Shutdown.String()

		if cmd.Flags().Changed(flagTimeoutAPIShutdown) {
			warnings = append(
				warnings,
				flagOverrideWarning(flagTimeoutAPIShutdown, parsed, c.config.timeout.apiShutdown),
			)
			logger.Debug("Flag overriding config value", "flag", flagTimeoutAPIShutdown,
				"config", parsed, "using", c.config.timeout.apiShutdown)
		} else {
			logger.Debug("Using config file value", "setting", "api.timeout.shutdown", "value", parsed)
			c.config.timeout.apiShutdown = parsed
		}
	}

	return warnings
}

// loadConfigMCP loads MCP configuration from config file, with flag overrides.
func (c *DaemonCmd) loadConfigMCP(mcp *config.MCPConfigSection, logger hclog.Logger, cmd *cobra.Command) []string {
	if mcp == nil {
		return nil
	}

	var warnings []string

	// Handle MCP timeout settings.
	if mcp.Timeout != nil {
		warnings = append(warnings, c.loadConfigMCPTimeout(mcp.Timeout, logger, cmd)...)
	}

	// Handle MCP interval settings.
	if mcp.Interval != nil {
		warnings = append(warnings, c.loadConfigMCPInterval(mcp.Interval, logger, cmd)...)
	}

	return warnings
}

// loadConfigMCPTimeout loads MCP timeout configuration from config file, with flag overrides.
func (c *DaemonCmd) loadConfigMCPTimeout(
	timeout *config.MCPTimeoutConfigSection,
	logger hclog.Logger,
	cmd *cobra.Command,
) []string {
	if timeout == nil {
		return nil
	}

	var warnings []string

	// Handle MCP init timeout.
	if timeout.Init != nil {
		parsed := timeout.Init.String()

		if cmd.Flags().Changed(flagTimeoutMCPInit) {
			warnings = append(
				warnings,
				flagOverrideWarning(flagTimeoutMCPInit, parsed, c.config.timeout.mcpInit),
			)
			logger.Debug("Flag overriding config value", "flag", flagTimeoutMCPInit,
				"config", parsed, "using", c.config.timeout.mcpInit)
		} else {
			logger.Debug("Using config file value", "setting", "mcp.timeout.init", "value", parsed)
			c.config.timeout.mcpInit = parsed
		}
	}

	// Handle MCP health check timeout.
	if timeout.Health != nil {
		parsed := timeout.Health.String()

		if cmd.Flags().Changed(flagTimeoutMCPHealth) {
			warnings = append(
				warnings,
				flagOverrideWarning(flagTimeoutMCPHealth, parsed, c.config.timeout.healthCheck),
			)
			logger.Debug("Flag overriding config value", "flag", flagTimeoutMCPHealth,
				"config", parsed, "using", c.config.timeout.healthCheck)
		} else {
			logger.Debug("Using config file value", "setting", "mcp.timeout.health", "value", parsed)
			c.config.timeout.healthCheck = parsed
		}
	}

	return warnings
}

// loadConfigMCPInterval loads MCP interval configuration from config file, with flag overrides.
func (c *DaemonCmd) loadConfigMCPInterval(
	interval *config.MCPIntervalConfigSection,
	logger hclog.Logger,
	cmd *cobra.Command,
) []string {
	if interval == nil || interval.Health == nil {
		return nil
	}

	parsed := interval.Health.String()

	var warnings []string
	if cmd.Flags().Changed(flagIntervalMCPHealth) {
		warnings = append(
			warnings,
			flagOverrideWarning(flagIntervalMCPHealth, parsed, c.config.interval.healthCheck),
		)
		logger.Debug("Flag overriding config value", "flag", flagIntervalMCPHealth,
			"config", parsed, "using", c.config.interval.healthCheck)
	} else {
		logger.Debug("Using config file value", "setting", "mcp.interval.health", "value", parsed)
		c.config.interval.healthCheck = parsed
	}

	return warnings
}

// loadConfigCORS loads CORS configuration from config file, with flag overrides.
func (c *DaemonCmd) loadConfigCORS(cors *config.CORSConfigSection, logger hclog.Logger, cmd *cobra.Command) []string {
	if cors == nil {
		return nil
	}

	var warnings []string

	// Handle CORS enable flag with correct precedence.
	if cors.Enable != nil {
		if cmd.Flags().Changed(flagCORSEnable) {
			// Flag explicitly set, it wins.
			warnings = append(warnings, flagOverrideWarning(flagCORSEnable, *cors.Enable, c.config.cors.enable))
			logger.Debug(
				"Flag overriding config value",
				"flag", flagCORSEnable,
				"config", *cors.Enable,
				"using", c.config.cors.enable,
			)
		} else {
			// Use config value.
			logger.Debug("Using config file value", "setting", "cors.enable", "value", *cors.Enable)
			c.config.cors.enable = *cors.Enable
		}
	}

	// Handle CORS origins.
	if len(cors.Origins) > 0 {
		if cmd.Flags().Changed(flagCORSOrigin) {
			warnings = append(warnings, flagOverrideWarning(flagCORSOrigin, cors.Origins, c.config.cors.origins))
			logger.Debug("Flag overriding config value", "flag", flagCORSOrigin,
				"config", cors.Origins, "using", c.config.cors.origins)
		} else {
			logger.Debug("Using config file value", "setting", "cors.origins", "value", cors.Origins)
			c.config.cors.origins = cors.Origins
		}
	}

	// Handle CORS methods.
	if len(cors.Methods) > 0 {
		if cmd.Flags().Changed(flagCORSMethod) {
			warnings = append(warnings, flagOverrideWarning(flagCORSMethod, cors.Methods, c.config.cors.methods))
			logger.Debug("Flag overriding config value", "flag", flagCORSMethod,
				"config", cors.Methods, "using", c.config.cors.methods)
		} else {
			logger.Debug("Using config file value", "setting", "cors.methods", "value", cors.Methods)
			c.config.cors.methods = cors.Methods
		}
	}

	// Handle CORS credentials.
	if cors.Credentials != nil {
		if cmd.Flags().Changed(flagCORSCredentials) {
			warnings = append(
				warnings,
				flagOverrideWarning(flagCORSCredentials, *cors.Credentials, c.config.cors.credentials),
			)
			logger.Debug("Flag overriding config value", "flag", flagCORSCredentials,
				"config", *cors.Credentials, "using", c.config.cors.credentials)
		} else {
			logger.Debug("Using config file value", "setting", "cors.credentials", "value", *cors.Credentials)
			c.config.cors.credentials = *cors.Credentials
		}
	}

	// Handle CORS headers.
	if len(cors.Headers) > 0 {
		if cmd.Flags().Changed(flagCORSHeaders) {
			warnings = append(warnings, flagOverrideWarning(flagCORSHeaders, cors.Headers, c.config.cors.headers))
			logger.Debug("Flag overriding config value", "flag", flagCORSHeaders,
				"config", cors.Headers, "using", c.config.cors.headers)
		} else {
			logger.Debug("Using config file value", "setting", "cors.headers", "value", cors.Headers)
			c.config.cors.headers = cors.Headers
		}
	}

	// Handle CORS expose headers.
	if len(cors.ExposeHeaders) > 0 {
		if cmd.Flags().Changed(flagCORSExposeHeaders) {
			warnings = append(
				warnings,
				flagOverrideWarning(flagCORSExposeHeaders, cors.ExposeHeaders, c.config.cors.exposedHeaders),
			)
			logger.Debug("Flag overriding config value", "flag", flagCORSExposeHeaders,
				"config", cors.ExposeHeaders, "using", c.config.cors.exposedHeaders)
		} else {
			logger.Debug("Using config file value", "setting", "cors.expose_headers", "value", cors.ExposeHeaders)
			c.config.cors.exposedHeaders = cors.ExposeHeaders
		}
	}

	// Handle CORS max age.
	if cors.MaxAge != nil {
		maxAgeStr := cors.MaxAge.String()
		if cmd.Flags().Changed(flagCORSMaxAge) {
			warnings = append(warnings, flagOverrideWarning(flagCORSMaxAge, maxAgeStr, c.config.cors.maxAge))
			logger.Debug("Flag overriding config value", "flag", flagCORSMaxAge,
				"config", maxAgeStr, "using", c.config.cors.maxAge)
		} else {
			logger.Debug("Using config file value", "setting", "cors.max_age", "value", maxAgeStr)
			c.config.cors.maxAge = maxAgeStr
		}
	}

	return warnings
}

// handleSignals processes OS signals for daemon lifecycle management.
// This function is intended to be called in a dedicated goroutine.
//
// For SIGHUP signals:
//  1. Attempts to set the shared reloading flag from false to true
//  2. If successful, sends to reloadChan for main loop to process
//  3. If flag already true, logs warning about duplicate reload requests and drops the signal
//  4. The main loop is responsible for resetting the flag after a reload is complete
//
// This coordination ensures only one reload runs at a time while allowing
// the actual reload work to happen in the main loop with proper context.
func (c *DaemonCmd) handleSignals(
	logger hclog.Logger,
	sigChan <-chan os.Signal,
	state *reloadState,
	shutdownCancel context.CancelFunc,
) {
	for sig := range sigChan {
		switch sig {
		case syscall.SIGHUP:
			if !state.reloading.CompareAndSwap(false, true) {
				logger.Warn("SIGHUP: reload already in progress, skipping")
				continue
			}
			select {
			case state.reloadChan <- struct{}{}:
				logger.Info("SIGHUP received, triggering reload")
			default:
				logger.Warn("SIGHUP: reload channel full, skipping")
			}
		case os.Interrupt, syscall.SIGTERM, syscall.SIGINT:
			logger.Info("Received shutdown signal", "signal", sig)
			shutdownCancel()
			return
		}
	}
}

// reloadServers reloads server configuration from config files.
// This method only reloads runtime servers; daemon config changes require a restart.
func (c *DaemonCmd) reloadServers(ctx context.Context, d *daemon.Daemon) error {
	cfg, err := c.LoadConfig(c.cfgLoader)
	if err != nil {
		return fmt.Errorf("%w: %w", config.ErrConfigLoadFailed, err)
	}

	execCtx, err := c.ctxLoader.Load(flags.RuntimeFile)
	if err != nil {
		return fmt.Errorf("failed to load runtime context: %w", err)
	}

	newServers, err := runtime.AggregateConfigs(cfg, execCtx)
	if err != nil {
		return fmt.Errorf("failed to aggregate configs: %w", err)
	}

	// Reload the servers in the daemon.
	if err := d.ReloadServers(ctx, newServers); err != nil {
		return fmt.Errorf("failed to reload servers: %w", err)
	}

	return nil
}

// validateFlags validates the command flags and their relationships.
func (c *DaemonCmd) validateFlags(cmd *cobra.Command) error {
	// Validate that other CORS flags require --cors-enable.
	// NOTE: --cors-origin is already handled by MarkFlagsRequiredTogether during flag definition.
	if !c.config.cors.enable {
		corsFlags := []struct {
			flag  string
			isSet bool
		}{
			{flag: flagCORSMethod, isSet: len(c.config.cors.methods) > 0},
			{flag: flagCORSHeaders, isSet: len(c.config.cors.headers) > 0},
			{flag: flagCORSExposeHeaders, isSet: len(c.config.cors.exposedHeaders) > 0},
			{flag: flagCORSCredentials, isSet: c.config.cors.credentials},
			{flag: flagCORSMaxAge, isSet: c.config.cors.maxAge != ""},
		}

		for _, o := range corsFlags {
			if cmd.Flags().Changed(o.flag) && o.isSet {
				return fmt.Errorf("--%s requires --%s", o.flag, flagCORSEnable)
			}
		}
	}

	// Validate duration strings.
	if c.config.cors.maxAge != "" {
		if _, err := time.ParseDuration(c.config.cors.maxAge); err != nil {
			return fmt.Errorf("invalid --%s duration: %w", flagCORSMaxAge, err)
		}
	}

	if c.config.timeout.apiShutdown != "" {
		if _, err := time.ParseDuration(c.config.timeout.apiShutdown); err != nil {
			return fmt.Errorf("invalid --%s duration: %w", flagTimeoutAPIShutdown, err)
		}
	}

	if c.config.timeout.mcpInit != "" {
		if _, err := time.ParseDuration(c.config.timeout.mcpInit); err != nil {
			return fmt.Errorf("invalid --%s duration: %w", flagTimeoutMCPInit, err)
		}
	}

	if c.config.timeout.healthCheck != "" {
		if _, err := time.ParseDuration(c.config.timeout.healthCheck); err != nil {
			return fmt.Errorf("invalid --%s duration: %w", flagTimeoutMCPHealth, err)
		}
	}

	if c.config.timeout.mcpShutdown != "" {
		if _, err := time.ParseDuration(c.config.timeout.mcpShutdown); err != nil {
			return fmt.Errorf("invalid --%s duration: %w", flagTimeoutMCPShutdown, err)
		}
	}

	if c.config.interval.healthCheck != "" {
		if _, err := time.ParseDuration(c.config.interval.healthCheck); err != nil {
			return fmt.Errorf("invalid --%s duration: %w", flagIntervalMCPHealth, err)
		}
	}

	return nil
}

// buildAPIOptions creates daemon API options from flags set on DaemonCmd.
func (c *DaemonCmd) buildAPIOptions() ([]daemon.APIOption, error) {
	var apiOpts []daemon.APIOption

	// Add CORS configuration if enabled.
	if c.config.cors.enable {
		apiOpts = append(apiOpts, daemon.WithCORSEnabled(true))
		apiOpts = append(apiOpts, daemon.WithCORSAllowOrigins(c.config.cors.origins))

		// Set methods if provided.
		if v := c.config.cors.methods; len(v) > 0 {
			apiOpts = append(apiOpts, daemon.WithCORSAllowMethods(v))
		}

		// Set request headers if provided.
		if v := c.config.cors.headers; len(v) > 0 {
			apiOpts = append(apiOpts, daemon.WithCORSAllowHeaders(v))
		}

		// Set exposed response headers if provided.
		if v := c.config.cors.exposedHeaders; len(v) > 0 {
			apiOpts = append(apiOpts, daemon.WithCORSExposeHeaders(v))
		}

		// Set credentials if different from default.
		if v := c.config.cors.credentials; v != daemon.DefaultCORSAllowCredentials() {
			apiOpts = append(apiOpts, daemon.WithCORSAllowCredentials(v))
		}

		// Parse and set max age duration.
		if v := c.config.cors.maxAge; v != "" {
			maxAge, err := time.ParseDuration(v)
			if err != nil {
				return nil, fmt.Errorf("invalid %s: %w", flagCORSMaxAge, err)
			}
			apiOpts = append(apiOpts, daemon.WithCORSMaxAge(maxAge))
		}
	}

	// Add API shutdown timeout if specified.
	if c.config.timeout.apiShutdown != "" {
		timeout, err := time.ParseDuration(c.config.timeout.apiShutdown)
		if err != nil {
			return nil, fmt.Errorf("invalid %s: %w", flagTimeoutAPIShutdown, err)
		}
		apiOpts = append(apiOpts, daemon.WithShutdownTimeout(timeout))
	}

	return apiOpts, nil
}

// buildDaemonOptions creates daemon options from command flags and API options.
func (c *DaemonCmd) buildDaemonOptions(apiOptions []daemon.APIOption) ([]daemon.Option, error) {
	daemonOpts := []daemon.Option{
		daemon.WithAPIOptions(apiOptions...),
	}

	// Add MCP server initialization timeout.
	if c.config.timeout.mcpInit != "" {
		timeout, err := time.ParseDuration(c.config.timeout.mcpInit)
		if err != nil {
			return nil, fmt.Errorf("invalid %s: %w", flagTimeoutMCPInit, err)
		}
		daemonOpts = append(daemonOpts, daemon.WithMCPServerInitTimeout(timeout))
	}

	// Add health check timeout.
	if c.config.timeout.healthCheck != "" {
		timeout, err := time.ParseDuration(c.config.timeout.healthCheck)
		if err != nil {
			return nil, fmt.Errorf("invalid %s: %w", flagTimeoutMCPHealth, err)
		}
		daemonOpts = append(daemonOpts, daemon.WithMCPServerHealthCheckTimeout(timeout))
	}

	// Add health check interval.
	if c.config.interval.healthCheck != "" {
		interval, err := time.ParseDuration(c.config.interval.healthCheck)
		if err != nil {
			return nil, fmt.Errorf("invalid %s: %w", flagIntervalMCPHealth, err)
		}
		daemonOpts = append(daemonOpts, daemon.WithMCPServerHealthCheckInterval(interval))
	}

	return daemonOpts, nil
}

// printDevBanner prints the development mode banner with comprehensive configuration info.
func (c *DaemonCmd) printDevBanner(w io.Writer, logger hclog.Logger, addr string) {
	logger.Info("Launching daemon in dev mode", "addr", addr)

	banner := fmt.Sprintf("mcpd daemon running in 'dev' mode.\n\n"+
		"  Local API:\thttp://%s/api/v1\n"+
		"  OpenAPI UI:\thttp://%s/docs\n"+
		"  Config file:\t%s\n"+
		"  Secrets file:\t%s\n",
		addr, addr, flags.ConfigFile, flags.RuntimeFile)

	if flags.LogPath != "" {
		banner += fmt.Sprintf("  Log file:\t%s => (%s)\n", flags.LogPath, flags.LogLevel)
	}

	banner += c.formatConfigInfo(addr)

	banner += "\nPress Ctrl+C to stop.\n\n"
	_, _ = fmt.Fprint(w, banner)
}

// formatConfigInfo formats all active configuration flags for display.
func (c *DaemonCmd) formatConfigInfo(addr string) string {
	var info strings.Builder

	// API configuration (only show if non-default).
	if addr != "" && addr != "0.0.0.0:8090" {
		info.WriteString(fmt.Sprintf("  API address:\t%s\n", addr))
	}

	// CORS configuration.
	if c.config.cors.enable {
		info.WriteString(fmt.Sprintf("  CORS enabled:\t%v (origins: %s)\n",
			c.config.cors.enable,
			strings.Join(c.config.cors.origins, ", ")))

		if len(c.config.cors.methods) > 0 {
			info.WriteString(fmt.Sprintf("  CORS methods:\t%s\n",
				strings.Join(c.config.cors.methods, ", ")))
		}

		if c.config.cors.credentials {
			info.WriteString("  CORS credentials:\ttrue\n")
		}

		if c.config.cors.maxAge != "" {
			info.WriteString(fmt.Sprintf("  CORS max age:\t%s\n", c.config.cors.maxAge))
		}
	}

	// Timeout configuration.
	if v := c.config.timeout.apiShutdown; v != "" && v != daemon.DefaultAPIShutdownTimeout().String() {
		info.WriteString(fmt.Sprintf("  API shutdown timeout:\t%s\n", v))
	}
	if v := c.config.timeout.mcpInit; v != "" && v != daemon.DefaultClientInitTimeout().String() {
		info.WriteString(fmt.Sprintf("  MCP init timeout:\t%s\n", v))
	}

	if v := c.config.timeout.healthCheck; v != "" && v != daemon.DefaultHealthCheckTimeout().String() {
		info.WriteString(fmt.Sprintf("  MCP health check timeout:\t%s\n", v))
	}

	if v := c.config.interval.healthCheck; v != "" && v != daemon.DefaultHealthCheckInterval().String() {
		info.WriteString(fmt.Sprintf("  MCP health check interval:\t%s\n", v))
	}

	return info.String()
}
