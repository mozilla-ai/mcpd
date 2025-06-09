package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd-cli/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd-cli/v2/internal/config"
	"github.com/mozilla-ai/mcpd-cli/v2/internal/context"
)

type SetArgsCmd struct {
	*cmd.BaseCmd
	Args []string
}

func NewSetArgsCmd(logger hclog.Logger) *cobra.Command {
	c := &SetArgsCmd{
		BaseCmd: &cmd.BaseCmd{Logger: logger},
	}

	cobraCmd := &cobra.Command{
		Use:   "set-args <server-name> --arg [--arg ...]",
		Short: "Set startup arguments for an MCP server.",
		Long:  c.longDescription(),
		RunE:  c.run,
	}

	cobraCmd.Flags().StringArrayVar(
		&c.Args,
		"arg",
		nil,
		"Specify startup argument for the server (can be repeated). Supports flags with or without values, e.g. --flag or --key=value.",
	)

	return cobraCmd
}

func (c *SetArgsCmd) longDescription() string {
	return `Set or update startup arguments for a specified MCP server in the runtime context configuration file (~/.mcpd/secrets.dev.toml).`
}

func (c *SetArgsCmd) run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("server name is required and cannot be empty")
	}
	serverName := strings.TrimSpace(args[0])

	normalizedArgs := config.NormalizeArgs(c.Args)

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
	serverCtx.Args = normalizedArgs
	if serverCtx.Env == nil {
		serverCtx.Env = map[string]string{}
	}
	cfg.Servers[serverName] = serverCtx

	if err := context.SaveExecutionContextConfig(filePath, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Startup arguments set for server '%s': %v\n", serverName, normalizedArgs)
	return nil
}
