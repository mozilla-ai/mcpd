package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
	"github.com/mozilla-ai/mcpd/v2/internal/daemon"
	"github.com/mozilla-ai/mcpd/v2/internal/flags"
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
		Short: "Launches an mcpd daemon instance",
		Long:  "Launches an mcpd daemon instance, which starts MCP servers and provides routing via HTTP API",
		RunE:  c.run,
	}

	cobraCommand.Flags().BoolVar(
		&c.Dev,
		"dev",
		false,
		"Run the daemon in development-focused mode",
	)

	cobraCommand.Flags().StringVar(
		&c.Addr,
		"addr",
		"localhost:8090",
		"Address for the daemon to bind (not applicable in --dev mode)",
	)
	cobraCommand.MarkFlagsMutuallyExclusive("dev", "addr")

	return cobraCommand, nil
}

// run is configured (via NewDaemonCmd) to be called by the Cobra framework when the command is executed.
// It may return an error (or nil, when there is no error).
func (c *DaemonCmd) run(cmd *cobra.Command, args []string) error {
	// Validate flags.
	addr := strings.TrimSpace(c.Addr)
	if err := daemon.IsValidAddr(addr); err != nil {
		return err
	}

	logger, err := c.Logger()
	if err != nil {
		return err
	}

	d, err := daemon.NewDaemon(logger, c.cfgLoader, addr)
	if err != nil {
		return fmt.Errorf("failed to create mcpd daemon instance: %w", err)
	}

	// Create the signal handling context for the application.
	daemonCtx, daemonCtxCancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer daemonCtxCancel()

	runErr := make(chan error, 1)
	go func() {
		if err := d.StartAndManage(daemonCtx); err != nil && !errors.Is(err, context.Canceled) {
			runErr <- err
		}
		close(runErr)
	}()

	// Print --dev mode banner if required.
	if c.Dev {
		logger.Info("Launching daemon in dev mode", "addr", addr)
		fmt.Printf("mcpd daemon running in 'dev' mode.\n\n"+
			"  Local API:\thttp://%s/api/v1\n"+
			"  OpenAPI UI:\thttp://%s/docs\n"+
			"  Config file:\t%s\n"+
			"  Secrets file:\t%s\n\n"+
			"Press Ctrl+C to stop.\n\n", addr, addr, flags.ConfigFile, flags.RuntimeFile)
	}

	select {
	case <-daemonCtx.Done():
		logger.Info("Shutting down daemon")
		err := <-runErr // Wait for cleanup and deferred logging.
		return err      // Graceful Ctrl+C / SIGTERM.
	case err := <-runErr:
		logger.Error("error running daemon instance", "error", err)
		return err // Propagate daemon failure.
	}
}
