package args

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
	"github.com/mozilla-ai/mcpd/v2/internal/context"
	"github.com/mozilla-ai/mcpd/v2/internal/flags"
)

type SetCmd struct {
	*cmd.BaseCmd
	ctxLoader context.Loader
}

func NewSetCmd(baseCmd *cmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	opts, err := cmdopts.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &SetCmd{
		BaseCmd:   baseCmd,
		ctxLoader: opts.ContextLoader,
	}

	cobraCmd := &cobra.Command{
		Use:   "set <server-name> -- --arg=value [--arg=value ...]",
		Short: "Set startup command line arguments for an MCP server",
		Long: "Set startup command line arguments for an MCP server in the " +
			"runtime context configuration file (e.g. `~/.config/mcpd/secrets.dev.toml`)",
		RunE: c.run,
		Args: func(cmd *cobra.Command, args []string) error {
			if cmd.ArgsLenAtDash() < 1 || strings.TrimSpace(args[0]) == "" {
				return fmt.Errorf("server-name is required")
			} else if cmd.ArgsLenAtDash() > 1 {
				return fmt.Errorf("too many arguments")
			} else if len(args) < 2 {
				return fmt.Errorf("argument(s) are required")
			}
			return nil
		},
	}

	return cobraCmd, nil
}

func (c *SetCmd) run(cmd *cobra.Command, args []string) error {
	serverName := strings.TrimSpace(args[0])
	if serverName == "" {
		return fmt.Errorf("server-name is required")
	}

	normalizedArgs := config.ProcessAllArgs(args[1:])
	cfg, err := c.ctxLoader.Load(flags.RuntimeFile)
	if err != nil {
		return fmt.Errorf("failed to load execution context config: %w", err)
	}

	server, exists := cfg.Get(serverName)
	if !exists {
		server.Name = serverName
	}

	newArgs := config.MergeArgs(server.Args, normalizedArgs)

	// Update...
	server.Args = newArgs
	if len(server.Env) == 0 {
		server.Env = map[string]string{}
	}

	res, err := cfg.Upsert(server)
	if err != nil {
		return fmt.Errorf("error setting arguments for server '%s': %w", serverName, err)
	}

	if _, err := fmt.Fprintf(
		cmd.OutOrStdout(),
		"✓ Startup arguments set for server '%s' (operation: %s): %v\n", serverName, string(res), normalizedArgs,
	); err != nil {
		return err
	}

	return nil
}
