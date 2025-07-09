package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
	"github.com/mozilla-ai/mcpd/v2/internal/daemon"
)

// DaemonCmd should be used to represent the 'daemon' command.
type DaemonCmd struct {
	*cmd.BaseCmd
	Dev       bool
	Addr      string
	cfgLoader config.Loader
}

// NewDaemonCmd creates a newly configured (Cobra) command.
func NewDaemonCmd(baseCmd *cmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	opts, err := cmdopts.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &DaemonCmd{
		BaseCmd:   baseCmd,
		cfgLoader: opts.ConfigLoader,
	}

	cobraCommand := &cobra.Command{
		Use:   "daemon",
		Short: "Launches an mcpd daemon instance (Execution Plane).",
		Long:  c.longDescription(),
		RunE:  c.run,
	}

	cobraCommand.Flags().BoolVar(
		&c.Dev,
		"dev",
		false,
		"Run the daemon in development-focused mode.",
	)

	cobraCommand.Flags().StringVar(
		&c.Addr,
		"addr",
		"localhost:8090",
		"Specify the address for the daemon to bind to (e.g., 'localhost:8090'). Not applicable in --dev mode.",
	)
	cobraCommand.MarkFlagsMutuallyExclusive("dev", "addr")

	return cobraCommand, nil
}

// longDescription returns the long version of the command description.
func (c *DaemonCmd) longDescription() string {
	return `Launches an mcpd daemon instance (Execution Plane).
In dev mode, binds to localhost, logs to console, and exposes local endpoint.
In prod, binds to 0.0.0.0, logs to stdout, and runs as background service.`
}

// run is configured (via NewDaemonCmd) to be called by the Cobra framework when the command is executed.
// It may return an error (or nil, when there is no error).
func (c *DaemonCmd) run(cmd *cobra.Command, args []string) error {
	// Validate flags.
	addr := strings.TrimSpace(c.Addr)
	if err := daemon.IsValidAddr(addr); err != nil {
		return fmt.Errorf("invalid address flag value: %s: %w", addr, err)
	}

	// TODO: Currently only runs in 'dev' mode... (even without flag)
	//	addr := "localhost:8080"
	//	if c.Addr != "" {
	//		addr = c.Addr
	//	}
	//
	//	c.Logger.Info("Launching daemon (dev mode)", "bindAddr", addr)
	//	c.Logger.Info("Local endpoint", "url", "http://"+addr+"/api")
	//	c.Logger.Info("Dev API key", "value", "dev-api-key-12345")   // TODO: Generate local key
	//	c.Logger.Info("Secrets file", "path", "~/.config/mcpd/secrets.dev.toml") // TODO: Configurable?
	//	c.Logger.Info("Press Ctrl+C to stop.")
	logger, err := c.Logger()
	if err != nil {
		return err
	}

	daemonCtx, daemonCtxCancel := context.WithCancel(context.Background())
	defer daemonCtxCancel()

	d, err := daemon.NewDaemon(logger, c.cfgLoader, addr)
	if err != nil {
		return fmt.Errorf("failed to create mcpd daemon instance: %w", err)
	}
	if err := d.StartAndManage(daemonCtx); err != nil {
		return fmt.Errorf("daemon start failed: %w", err)
	}

	return nil
}
