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

type SetCmd struct {
	*cmd.BaseCmd
	ctxLoader context.Loader
}

func NewSetCmd(baseCmd *cmd.BaseCmd, opt ...options.CmdOption) (*cobra.Command, error) {
	opts, err := options.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &SetCmd{
		BaseCmd:   baseCmd,
		ctxLoader: opts.ContextLoader,
	}

	cobraCmd := &cobra.Command{
		Use:   "set <server-name> KEY=VALUE [KEY=VALUE ...]",
		Short: "Set or update environment variables for an MCP server",
		Long: "Set or update environment variables for a specified MCP server in the " +
			"runtime context configuration file (e.g. `~/.config/mcpd/secrets.dev.toml`)",
		RunE: c.run,
		Args: cobra.MinimumNArgs(2), // server_name and KEY=VALUE
	}

	return cobraCmd, nil
}

func (c *SetCmd) run(cmd *cobra.Command, args []string) error {
	serverName := strings.TrimSpace(args[0])
	if serverName == "" {
		return fmt.Errorf("server-name is required")
	}

	envVars := args[1:]
	envMap := make(map[string]string, len(envVars))
	for _, kvp := range envVars {
		parts := strings.SplitN(kvp, "=", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
			return fmt.Errorf("invalid environment variable format: '%s', expected KEY=VALUE", kvp)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		envMap[key] = value
	}

	cfg, err := c.ctxLoader.Load(flags.RuntimeFile)
	if err != nil {
		return fmt.Errorf("failed to load execution context config: %w", err)
	}

	server, exists := cfg.Get(serverName)
	if !exists {
		server.Name = serverName
	}

	// Ensure the map is initialized, in case it didn't exist before.
	if server.Env == nil {
		server.Env = map[string]string{}
	}

	// Merge or overwrite environment variables
	for k, v := range envMap {
		server.Env[k] = v
	}

	res, err := cfg.Upsert(server)
	if err != nil {
		return fmt.Errorf("error setting environment variables for server '%s': %w", serverName, err)
	}

	if _, err := fmt.Fprintf(
		cmd.OutOrStdout(),
		"✓ Environment variables set for server '%s' (operation: %s): %v\n", serverName, string(res), slices.Collect(maps.Keys(envMap)),
	); err != nil {
		return err
	}

	return nil
}
