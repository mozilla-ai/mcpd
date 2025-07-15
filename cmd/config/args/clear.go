package args

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/context"
	"github.com/mozilla-ai/mcpd/v2/internal/flags"
)

type ClearCmd struct {
	*cmd.BaseCmd
	Force     bool
	ctxLoader context.Loader
}

func NewClearCmd(baseCmd *cmd.BaseCmd, opt ...options.CmdOption) (*cobra.Command, error) {
	opts, err := options.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &ClearCmd{
		BaseCmd:   baseCmd,
		ctxLoader: opts.ContextLoader,
	}

	cobraCmd := &cobra.Command{
		Use:   "clear <server-name>",
		Short: "Clears configured command line arguments (flags) for an MCP server",
		Long: "Clears configured command line arguments (flags) for an MCP server, " +
			"from the runtime context configuration file (e.g. ~/.config/mcpd/secrets.dev.toml)",
		RunE: c.run,
		Args: cobra.MinimumNArgs(1), // server-name
	}

	cobraCmd.Flags().BoolVar(
		&c.Force,
		"force",
		false,
		"Force clearing of all command line arguments for the specified server without confirmation",
	)

	return cobraCmd, nil
}

func (c *ClearCmd) run(cmd *cobra.Command, args []string) error {
	serverName := strings.TrimSpace(args[0])
	if serverName == "" {
		return fmt.Errorf("server-name is required")
	}

	if !c.Force {
		return fmt.Errorf("this is a destructive operation. To clear all command line arguments for '%s', "+
			"please re-run the command with the --force flag", serverName)
	}

	cfg, err := c.ctxLoader.Load(flags.RuntimeFile)
	if err != nil {
		return fmt.Errorf("failed to load execution context config: %w", err)
	}

	if s, ok := cfg.ListServers()[serverName]; ok {
		if err := cfg.RemoveServer(serverName); err != nil {
			return fmt.Errorf("error removing server, failed to clear argument config for '%s': %w", serverName, err)
		}

		s.Args = []string{}

		if err := cfg.AddServer(s); err != nil {
			return fmt.Errorf("error re-adding server, failed to clear argument config for '%s': %w", serverName, err)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Arguments cleared for server '%s'\n", serverName)

	return nil
}
