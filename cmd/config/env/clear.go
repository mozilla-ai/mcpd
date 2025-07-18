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
		Short: "Clears configured environment variables for an MCP server",
		Long: "Clears environment variables for a specified MCP server from the " +
			"runtime context configuration file (e.g. `~/.config/mcpd/secrets.dev.toml`)",
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

	cfg, err := c.ctxLoader.Load(flags.RuntimeFile)
	if err != nil {
		return fmt.Errorf("failed to load execution context config: %w", err)
	}

	s, ok := cfg.Get(serverName)
	if !ok {
		return fmt.Errorf("server '%s' not found in configuration", serverName)
	}

	s.Env = make(map[string]string)
	res, err := cfg.Upsert(s)
	if err != nil {
		return fmt.Errorf("error clearing environment variables for server '%s': %w", serverName, err)
	}

	if _, err := fmt.Fprintf(
		cmd.OutOrStdout(),
		"✓ Environment variables cleared for server '%s' (operation: %s)\n", serverName, string(res),
	); err != nil {
		return err
	}

	return nil
}
