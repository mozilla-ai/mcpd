package env

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/context"
	"github.com/mozilla-ai/mcpd/v2/internal/flags"
)

type RemoveCmd struct {
	*cmd.BaseCmd
	EnvVars   []string
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

	// mcpd config env remove time KEY [KEY ...]
	cobraCmd := &cobra.Command{
		Use:   "remove <server-name> KEY [KEY ...]",
		Short: "Remove environment variables for an MCP server",
		Long: "Remove environment variables for a specified MCP server in the " +
			"runtime context configuration file (e.g. `~/.config/mcpd/secrets.dev.toml`)",
		RunE: c.run,
		Args: cobra.MinimumNArgs(2), // server-name + KEY ...
	}

	return cobraCmd, nil
}

func (c *RemoveCmd) run(cmd *cobra.Command, args []string) error {
	serverName := strings.TrimSpace(args[0])
	if serverName == "" {
		return fmt.Errorf("server-name is required")
	}

	envVars := args[1:]
	envMap := make(map[string]struct{}, len(envVars))
	for _, key := range envVars {
		key = strings.TrimSpace(key)
		envMap[key] = struct{}{}
	}

	cfg, err := c.ctxLoader.Load(flags.RuntimeFile)
	if err != nil {
		return fmt.Errorf("failed to load execution context config: %w", err)
	}

	serverCtx, ok := cfg.Get(serverName)
	if !ok {
		return fmt.Errorf("server '%s' not found in configuration", serverName)
	}

	for key := range envMap {
		delete(serverCtx.Env, key)
	}
	res, err := cfg.Upsert(serverCtx)
	if err != nil {
		return fmt.Errorf("error removing environment variables for server '%s': %w", serverName, err)
	}

	if _, err := fmt.Fprintf(
		cmd.OutOrStdout(),
		"✓ Environment variables removed for server '%s' (operation: %s): %v\n", serverName, string(res), slices.Collect(maps.Keys(envMap)),
	); err != nil {
		return err
	}

	return nil
}
