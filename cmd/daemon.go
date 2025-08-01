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
	configcontext "github.com/mozilla-ai/mcpd/v2/internal/context"
	"github.com/mozilla-ai/mcpd/v2/internal/daemon"
	"github.com/mozilla-ai/mcpd/v2/internal/flags"
)

// DaemonCmd should be used to represent the 'daemon' command.
type DaemonCmd struct {
	*cmd.BaseCmd
	Dev       bool
	Addr      string
	cfgLoader config.Loader
	ctxLoader configcontext.Loader
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
		ctxLoader: opts.ContextLoader,
	}

	cobraCommand := &cobra.Command{
		Use:   "daemon [--dev] [--addr]",
		Short: "Launches an `mcpd` daemon instance",
		Long:  "Launches an `mcpd` daemon instance, which starts MCP servers and provides routing via HTTP API",
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
		"0.0.0.0:8090",
		"Address for the daemon to bind (not applicable in --dev mode)",
	)

	cobraCommand.MarkFlagsMutuallyExclusive("dev", "addr")

	return cobraCommand, nil
}

// run is configured (via NewDaemonCmd) to be called by the Cobra framework when the command is executed.
// It may return an error (or nil, when there is no error).
func (c *DaemonCmd) run(_ *cobra.Command, _ []string) error {
	logger, err := c.Logger()
	if err != nil {
		return err
	}

	// Validate flags.
	addr := strings.TrimSpace(c.Addr)

	// Override address for dev mode.
	if c.Dev {
		devAddr := "localhost:8090"
		logger.Info("Development-focused mode", "addr", addr, "override", devAddr)
		addr = devAddr
	}

	if err := daemon.IsValidAddr(addr); err != nil {
		return err
	}

	opts, err := daemon.NewDaemonOpts(logger, c.cfgLoader, c.ctxLoader)
	if err != nil {
		return fmt.Errorf("error configuring mcpd daemon options: %w", err)
	}
	d, err := daemon.NewDaemon(addr, opts)
	if err != nil {
		return fmt.Errorf("failed to create mcpd daemon instance: %w", err)
	}

	// Create the signal handling context for the application.
	daemonCtx, daemonCtxCancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM, syscall.SIGINT,
	)
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
		banner := fmt.Sprintf("mcpd daemon running in 'dev' mode.\n\n"+
			"  Local API:\thttp://%s/api/v1\n"+
			"  OpenAPI UI:\thttp://%s/docs\n"+
			"  Config file:\t%s\n"+
			"  Secrets file:\t%s\n",
			addr, addr, flags.ConfigFile, flags.RuntimeFile)

		if flags.LogPath != "" {
			banner += fmt.Sprintf("  Log file:\t%s => (%s)\n", flags.LogPath, flags.LogLevel)
		}

		banner += "\nPress Ctrl+C to stop.\n\n"
		fmt.Print(banner)
	}

	select {
	case <-daemonCtx.Done():
		logger.Info("Shutting down daemon")
		err := <-runErr // Wait for cleanup and deferred logging.
		return err      // Graceful Ctrl+C / SIGTERM.
	case err := <-runErr:
		logger.Error("daemon exited with error", "error", err)
		return err // Propagate daemon failure.
	}
}
