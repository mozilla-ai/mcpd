package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	internalcmd "github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
)

// InspectorCmd represents the 'inspector' command.
type InspectorCmd struct {
	*internalcmd.BaseCmd
}

// NewInspectorCmd creates a newly configured (Cobra) command.
func NewInspectorCmd(baseCmd *internalcmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	c := &InspectorCmd{
		BaseCmd: baseCmd,
	}

	cobraCommand := &cobra.Command{
		Use:   "inspector [command] [args]",
		Short: "Start the MCP inspector tool",
		Long: "Start the MCP inspector tool via npx for quickly testing MCP servers. " +
			"Optionally pass your desired command and arguments to the inspector. " +
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
	npxArgs := append([]string{"@modelcontextprotocol/inspector"}, args...)
	npxCommand := exec.CommandContext(ctx, "npx", npxArgs...)
	npxCommand.Stdout = os.Stdout
	npxCommand.Stderr = os.Stderr

	fmt.Printf("Starting the MCP inspector: npx %s...\n", strings.Join(npxArgs, " "))
	err := npxCommand.Start()
	if err != nil {
		return fmt.Errorf("failed to start the inspector: %w", err)
	}

	fmt.Printf("Press Ctrl+C to stop.\n")
	err = npxCommand.Wait()

	if ctx.Err() == context.Canceled {
		// Graceful shutdown with Ctrl+C (or SIGTERM)
		fmt.Printf("\nShutting down the inspector...\n")
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to run the inspector: %w", err)
	}

	return nil
}
