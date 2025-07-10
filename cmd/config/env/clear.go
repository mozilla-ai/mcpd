package env

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
	Force bool
}

func NewClearCmd(baseCmd *cmd.BaseCmd, _ ...options.CmdOption) (*cobra.Command, error) {
	c := &ClearCmd{
		BaseCmd: baseCmd,
	}

	cobraCmd := &cobra.Command{
		Use:   "clear <server-name>",
		Short: "Clears configured environment variables for an MCP server",
		Long: "Clears environment variables for a specified MCP server from the " +
			"runtime context configuration file (e.g. ~/.config/mcpd/secrets.dev.toml)",
		RunE: c.run,
		Args: cobra.MinimumNArgs(1), // server-name
	}

	cobraCmd.Flags().BoolVar(
		&c.Force,
		"force",
		false,
		"Force clearing of all environment variables for the specified server without confirmation",
	)

	return cobraCmd, nil
}

func (c *ClearCmd) run(cmd *cobra.Command, args []string) error {
	serverName := strings.TrimSpace(args[0])
	if serverName == "" {
		return fmt.Errorf("server-name is required")
	}

	if !c.Force {
		return fmt.Errorf("this is a destructive operation. To clear all environment variables for '%s', "+
			"please re-run the command with the --force flag", serverName)
	}

	cfg, err := context.LoadExecutionContextConfig(flags.RuntimeFile)
	if err != nil {
		return fmt.Errorf("failed to load execution context config: %w", err)
	}

	if s, ok := cfg.Servers[serverName]; ok {
		// Clear the env map and reassign the server in the config.
		s.Env = make(map[string]string)
		cfg.Servers[serverName] = s
		if err := context.SaveExecutionContextConfig(flags.RuntimeFile, cfg); err != nil {
			return fmt.Errorf("failed to clear env var config for '%s': %w", serverName, err)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Environment variables cleared for server '%s'\n", serverName)

	return nil
}
