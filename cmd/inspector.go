package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	internalcmd "github.com/mozilla-ai/mcpd/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/internal/runtime"
)

// InspectorCmd represents the 'inspector' command.
type InspectorCmd struct {
	*internalcmd.BaseCmd
}

// NewInspectorCmd creates a newly configured (Cobra) command.
func NewInspectorCmd(baseCmd *internalcmd.BaseCmd, _ ...cmdopts.CmdOption) (*cobra.Command, error) {
	c := &InspectorCmd{
		BaseCmd: baseCmd,
	}

	cobraCommand := &cobra.Command{
		Use:   "inspector [command] [args]",
		Short: "Start the MCP inspector tool",
		Long: "Start the MCP inspector tool via npx for quickly testing MCP servers. " +
			"Optionally pass your desired command and arguments to the inspector. " +
			"Note that the latest version of the inspector package is used (@modelcontextprotocol/inspector@latest). " +
			"For more information, see https://modelcontextprotocol.io/docs/tools/inspector.",
		RunE: c.run,
	}

	return cobraCommand, nil
}

// run is configured (via NewInspectorCmd) to be called by the Cobra framework when the command is executed.
// It may return an error (or nil, when there is no error, or there was a graceful shutdown).
func (c *InspectorCmd) run(cmd *cobra.Command, args []string) error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM, syscall.SIGINT,
	)
	defer stop()

	// Create a new npx process with the user provided arguments
	// Bind the process's stdout and stderr to our own for streaming output
	npxArgs := append([]string{"@modelcontextprotocol/inspector@latest"}, args...)
	npxCommand := exec.CommandContext(ctx, string(runtime.NPX), npxArgs...)
	npxCommand.Stdout = os.Stdout
	npxCommand.Stderr = os.Stderr

	_, _ = fmt.Fprintf(
		cmd.OutOrStdout(), "Starting the MCP inspector: npx %s...\n",
		strings.Join(npxArgs, " "),
	)
	err := npxCommand.Start()
	if err != nil {
		return fmt.Errorf("error running the inspector: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Press Ctrl+C to stop.\n")
	err = npxCommand.Wait()

	if errors.Is(ctx.Err(), context.Canceled) {
		// Graceful shutdown with Ctrl+C (or SIGTERM)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nShutting down the inspector...\n")
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to run the inspector: %w", err)
	}

	return nil
}
