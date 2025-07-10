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

type SetCmd struct {
	*cmd.BaseCmd
}

func NewSetCmd(baseCmd *cmd.BaseCmd, _ ...options.CmdOption) (*cobra.Command, error) {
	c := &SetCmd{
		BaseCmd: baseCmd,
	}

	cobraCmd := &cobra.Command{
		Use:   "set <server-name> KEY=VALUE [KEY=VALUE ...]",
		Short: "Set or update environment variables for an MCP server",
		Long: "Set or update environment variables for a specified MCP server in the " +
			"runtime context configuration file (e.g. ~/.config/mcpd/secrets.dev.toml)",
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

	cfg, err := context.LoadOrInitExecutionContext(flags.RuntimeFile)
	if err != nil {
		return fmt.Errorf("failed to load execution context config: %w", err)
	}

	serverCtx := cfg.Servers[serverName]
	if serverCtx.Env == nil {
		serverCtx.Env = map[string]string{}
	}

	// Merge or overwrite environment variables
	for k, v := range envMap {
		serverCtx.Env[k] = v
	}
	cfg.Servers[serverName] = serverCtx

	if err := context.SaveExecutionContextConfig(flags.RuntimeFile, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Environment variables set for server '%s': %v\n", serverName, envMap)
	return nil
}
