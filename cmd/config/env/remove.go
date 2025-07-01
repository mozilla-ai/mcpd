package env

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/context"
)

type RemoveCmd struct {
	*cmd.BaseCmd
	EnvVars []string
}

func NewRemoveCmd(baseCmd *cmd.BaseCmd, _ ...options.CmdOption) (*cobra.Command, error) {
	c := &RemoveCmd{
		BaseCmd: baseCmd,
	}

	// mcpd config env remove time KEY [KEY ...]
	cobraCmd := &cobra.Command{
		Use:   "remove <server-name> KEY [KEY ...]",
		Short: "Remove environment variables for an MCP server.",
		Long: `Remove environment variables for a specified MCP server in the runtime context configuration file 
		(e.g. ~/.mcpd/secrets.dev.toml).`,
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

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}
	filePath := filepath.Join(homeDir, ".mcpd", "secrets.dev.toml") // TODO: Allow configuration via flag

	cfg, err := context.LoadExecutionContextConfig(filePath)
	if err != nil {
		return fmt.Errorf("failed to load execution context config: %w", err)
	}

	if serverCtx := cfg.Servers[serverName]; serverCtx.Env != nil {
		evs := serverCtx.Env
		for key := range envMap {
			delete(evs, key)
		}
		cfg.Servers[serverName] = serverCtx

		if err := context.SaveExecutionContextConfig(filePath, cfg); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Environment variables removed for server '%s': %v\n", serverName, envMap)
	return nil
}
