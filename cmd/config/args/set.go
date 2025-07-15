package args

import (
	"fmt"
	"slices"
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
		Short: "Set startup command line arguments (flags) for an MCP server",
		Long: "Set startup command line arguments (flags) for an MCP server in the " +
			"runtime context configuration file (e.g. ~/.config/mcpd/secrets.dev.toml)",
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

	normalizedArgs := config.NormalizeArgs(args[1:])
	cfg, err := c.ctxLoader.Load(flags.RuntimeFile)
	if err != nil {
		return fmt.Errorf("failed to load execution context config: %w", err)
	}

	server, exists := cfg.ListServers()[serverName]
	if !exists {
		server.Name = serverName
	}

	newArgs := config.MergeArgs(server.Args, normalizedArgs)
	if !slices.Equal(newArgs, server.Args) {
		if exists {
			if err := cfg.RemoveServer(serverName); err != nil {
				return fmt.Errorf("error removing server, failed to set args in config for '%s': %w", serverName, err)
			}
		}

		// Update...
		server.Args = newArgs
		if len(server.Env) == 0 {
			server.Env = map[string]string{}
		}

		if err := cfg.AddServer(server); err != nil {
			return fmt.Errorf("error re-adding server, failed to set args in config for '%s': %w", serverName, err)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Startup arguments set for server '%s': %v\n", serverName, normalizedArgs)

	return nil
}
