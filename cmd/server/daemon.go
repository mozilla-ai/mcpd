package server

import (
	"context"
	"fmt"

	"github.com/mozilla-ai/mcpd-cli/v2/internal/daemon"
	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd-cli/v2/internal/cmd"
)

// DaemonCmd should be used to represent the 'daemon' command.
type DaemonCmd struct {
	*cmd.BaseCmd
	Dev  bool
	Addr string
}

// NewDaemonCmd creates a newly configured (Cobra) command.
func NewDaemonCmd(baseCmd *cmd.BaseCmd) *cobra.Command {
	c := &DaemonCmd{
		BaseCmd: baseCmd,
	}

	cobraCommand := &cobra.Command{
		Use:   "daemon",
		Short: "Launches an mcpd daemon instance (Execution Plane).",
		Long:  c.longDescription(),
		RunE:  c.run,
	}

	cobraCommand.Flags().BoolVar(&c.Dev, "dev", false, "Run the daemon in development-focused mode.")
	cobraCommand.Flags().StringVar(
		&c.Addr,
		"addr",
		"",
		"Specify the address for the daemon to bind to (e.g., 'localhost:8080'). Only applicable in --dev mode.",
	)
	cobraCommand.MarkFlagsMutuallyExclusive("dev", "addr")

	return cobraCommand
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
	//	addr := "localhost:8080"
	//	if c.Addr != "" {
	//		addr = c.Addr
	//	}
	//
	//	c.Logger.Info("Launching daemon (dev mode)", "bindAddr", addr)
	//	c.Logger.Info("Local endpoint", "url", "http://"+addr+"/api")
	//	c.Logger.Info("Dev API key", "value", "dev-api-key-12345")   // TODO: Generate local key
	//	c.Logger.Info("Secrets file", "path", "~/.mcpd/secrets.dev") // TODO: Configurable?
	//	c.Logger.Info("Press Ctrl+C to stop.")

	logger := c.Logger()
	d := daemon.NewDaemon(logger)
	if err := d.StartAndManage(context.Background()); err != nil {
		return fmt.Errorf("daemon start failed: %w", err)
	}

	return nil
}
