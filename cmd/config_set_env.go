package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/context"
)

type SetEnvCmd struct {
	*cmd.BaseCmd
	EnvVars []string
}

func NewSetEnvCmd(baseCmd *cmd.BaseCmd, _ ...cmdopts.CmdOption) (*cobra.Command, error) {
	c := &SetEnvCmd{
		BaseCmd: baseCmd,
	}

	cobraCmd := &cobra.Command{
		Use:   "set-env <server-name> --env KEY=VALUE [--env KEY=VALUE ...]",
		Short: "Set environment variables for an MCP server.",
		Long:  c.longDescription(),
		RunE:  c.run,
	}

	cobraCmd.Flags().StringArrayVar(
		&c.EnvVars,
		"env",
		nil,
		"Specify environment variable for the server (can be repeated). Format: KEY=VALUE.",
	)

	return cobraCmd, nil
}

func (c *SetEnvCmd) longDescription() string {
	return `Set or update environment variables for a specified MCP server in the runtime context configuration file (~/.mcpd/secrets.dev.toml).`
}

func (c *SetEnvCmd) run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("server name is required and cannot be empty")
	}
	serverName := strings.TrimSpace(args[0])

	envMap := map[string]string{}
	for _, env := range c.EnvVars {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
			return fmt.Errorf("invalid environment variable format: '%s', expected KEY=VALUE", env)
		}
		key := strings.TrimSpace(parts[0])
		value := parts[1]
		envMap[key] = value
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}
	filePath := filepath.Join(homeDir, ".mcpd", "secrets.dev.toml")

	cfg, err := context.LoadExecutionContextConfig(filePath)
	if err != nil {
		// If not exists, start with empty config
		cfg = context.ExecutionContextConfig{
			Servers: map[string]context.ServerExecutionContext{},
		}
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

	if err := context.SaveExecutionContextConfig(filePath, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Environment variables set for server '%s': %v\n", serverName, envMap)
	return nil
}
