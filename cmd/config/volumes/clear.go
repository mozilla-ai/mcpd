package volumes

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/internal/context"
	"github.com/mozilla-ai/mcpd/internal/flags"
)

// clearCmd handles clearing all volume mappings from an MCP server.
type clearCmd struct {
	*cmd.BaseCmd
	force     bool
	ctxLoader context.Loader
}

// NewClearCmd creates a new clear command for volume configuration.
func NewClearCmd(baseCmd *cmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	opts, err := cmdopts.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &clearCmd{
		BaseCmd:   baseCmd,
		ctxLoader: opts.ContextLoader,
	}

	cobraCmd := &cobra.Command{
		Use:   "clear <server-name>",
		Short: "Clear all volume mappings for a server",
		Long: "Clears all volume mappings for the specified server from the " +
			"runtime context configuration file (e.g. " + flags.RuntimeFile + ").\n\n" +
			"This is a destructive operation and requires the --force flag.\n\n" +
			"Examples:\n" +
			"  # Clear all volume mappings\n" +
			"  mcpd config volumes clear filesystem --force",
		RunE: c.run,
		Args: cobra.ExactArgs(1),
	}

	cobraCmd.Flags().BoolVar(
		&c.force,
		"force",
		false,
		"Force clearing of all volume mappings for the specified server without confirmation",
	)

	return cobraCmd, nil
}

// run executes the clear command, removing all volume mappings from the server config.
func (c *clearCmd) run(cobraCmd *cobra.Command, args []string) error {
	serverName := strings.TrimSpace(args[0])
	if serverName == "" {
		return fmt.Errorf("server-name is required")
	}

	if !c.force {
		return fmt.Errorf("this is a destructive operation. To clear all volumes for '%s', "+
			"please re-run the command with the --force flag", serverName)
	}

	cfg, err := c.ctxLoader.Load(flags.RuntimeFile)
	if err != nil {
		return fmt.Errorf("failed to load execution context config: %w", err)
	}

	server, ok := cfg.Get(serverName)
	if !ok {
		return fmt.Errorf("server '%s' not found in configuration", serverName)
	}

	server = withVolumes(server, context.VolumeExecutionContext{})

	res, err := cfg.Upsert(server)
	if err != nil {
		return fmt.Errorf("error clearing volumes for server '%s': %w", serverName, err)
	}

	if _, err := fmt.Fprintf(
		cobraCmd.OutOrStdout(),
		"✓ Volumes cleared for server '%s' (operation: %s)\n", serverName, string(res),
	); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	return nil
}
