package args

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
	"github.com/mozilla-ai/mcpd/v2/internal/context"
)

type RemoveCmd struct {
	*cmd.BaseCmd
}

func NewRemoveCmd(baseCmd *cmd.BaseCmd, _ ...options.CmdOption) (*cobra.Command, error) {
	c := &RemoveCmd{
		BaseCmd: baseCmd,
	}

	// mcpd config args remove time -- --arg [--arg ...]
	cobraCmd := &cobra.Command{
		Use:     "remove <server-name> -- --arg [--arg ...]",
		Example: "remove time -- --local-timezone",
		Short:   "Remove command line arguments (flags) for an MCP server.",
		Long: `Remove command line arguments (flags) for a specified MCP server in the runtime context configuration file
		(e.g. ~/.mcpd/secrets.dev.toml).`,
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

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}
	filePath := filepath.Join(homeDir, ".mcpd", "secrets.dev.toml") // TODO: Allow configuration via flag

	cfg, err := context.LoadExecutionContextConfig(filePath)
	if err != nil {
		return fmt.Errorf("failed to load execution context config: %w", err)
	}

	if serverCtx := cfg.Servers[serverName]; serverCtx.Args != nil {
		toRemove := slices.Collect(maps.Keys(argMap))
		filtered := config.RemoveMatchingFlags(serverCtx.Args, toRemove)

		// Only modify the file if there are actual changes to be made.
		if !slices.Equal(slices.Clone(serverCtx.Args), slices.Clone(filtered)) {
			// Update the args, and the server.
			serverCtx.Args = filtered
			cfg.Servers[serverName] = serverCtx

			if err := context.SaveExecutionContextConfig(filePath, cfg); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Args removed for server '%s': %v\n", serverName, argMap)
	return nil
}
