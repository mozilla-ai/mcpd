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
}

func NewSetCmd(baseCmd *cmd.BaseCmd, _ ...cmdopts.CmdOption) (*cobra.Command, error) {
	c := &SetCmd{
		BaseCmd: baseCmd,
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

	fmt.Fprintf(cmd.OutOrStdout(), "args: %#v\n", args)

	normalizedArgs := config.NormalizeArgs(args[1:])
	cfg, err := context.LoadOrInitExecutionContext(flags.RuntimeFile)
	if err != nil {
		return fmt.Errorf("failed to load execution context config: %w", err)
	}

	serverCtx := cfg.Servers[serverName]
	serverCtx.Args = config.MergeArgs(serverCtx.Args, normalizedArgs)
	if serverCtx.Env == nil {
		serverCtx.Env = map[string]string{}
	}
	cfg.Servers[serverName] = serverCtx

	if err := context.SaveExecutionContextConfig(flags.RuntimeFile, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Startup arguments set for server '%s': %v\n", serverName, normalizedArgs)

	return nil
}
