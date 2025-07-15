package args

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
	"github.com/mozilla-ai/mcpd/v2/internal/context"
	"github.com/mozilla-ai/mcpd/v2/internal/flags"
)

type RemoveCmd struct {
	*cmd.BaseCmd
	ctxLoader context.Loader
}

func NewRemoveCmd(baseCmd *cmd.BaseCmd, opt ...options.CmdOption) (*cobra.Command, error) {
	opts, err := options.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &RemoveCmd{
		BaseCmd:   baseCmd,
		ctxLoader: opts.ContextLoader,
	}

	// mcpd config args remove time -- --arg [--arg ...]
	cobraCmd := &cobra.Command{
		Use:     "remove <server-name> -- --arg [--arg ...]",
		Example: "remove time -- --local-timezone",
		Short:   "Remove command line arguments (flags) for an MCP server",
		Long: "Remove command line arguments (flags) for a specified MCP server in the " +
			"runtime context configuration file (e.g. ~/.config/mcpd/secrets.dev.toml)",
		RunE: c.run,
		Args: cobra.MinimumNArgs(2), // server-name + --arg ...
	}

	return cobraCmd, nil
}

func (c *RemoveCmd) run(cmd *cobra.Command, args []string) error {
	serverName := strings.TrimSpace(args[0])
	if serverName == "" {
		return fmt.Errorf("server-name is required")
	}

	argVars := args[1:]
	argMap := make(map[string]struct{}, len(argVars))
	for _, key := range argVars {
		key = strings.TrimSpace(key)
		argMap[key] = struct{}{}
	}

	cfg, err := c.ctxLoader.Load(flags.RuntimeFile)
	if err != nil {
		return fmt.Errorf("failed to load execution context config: %w", err)
	}

	if serverCtx, ok := cfg.ListServers()[serverName]; ok && serverCtx.Args != nil {
		toRemove := slices.Collect(maps.Keys(argMap))
		filtered := config.RemoveMatchingFlags(serverCtx.Args, toRemove)

		// Only modify the file if there are actual changes to be made.
		if !slices.Equal(slices.Clone(serverCtx.Args), slices.Clone(filtered)) {
			if err := cfg.RemoveServer(serverName); err != nil {
				return fmt.Errorf("error removing server, failed to remove args from config for '%s': %w", serverName, err)
			}

			// Update the args, and the server.
			serverCtx.Args = filtered

			if err := cfg.AddServer(serverCtx); err != nil {
				return fmt.Errorf("error re-adding server, failed to remove args from config for '%s': %w", serverName, err)
			}
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Args removed for server '%s': %v\n", serverName, slices.Collect(maps.Keys(argMap)))

	return nil
}
