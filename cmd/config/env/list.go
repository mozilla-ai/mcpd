package env

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/context"
	"github.com/mozilla-ai/mcpd/v2/internal/flags"
)

type ListCmd struct {
	*cmd.BaseCmd
}

func NewListCmd(baseCmd *cmd.BaseCmd, _ ...options.CmdOption) (*cobra.Command, error) {
	c := &ListCmd{
		BaseCmd: baseCmd,
	}

	cobraCmd := &cobra.Command{
		Use:   "list <server-name>",
		Short: "Lists configured environment variables for a specific MCP server.",
		Long: `Lists configured environment variables for a specific MCP server, using the runtime context configuration file 
		(e.g. ~/.config/mcpd/secrets.dev.toml).`,
		RunE: c.run,
		Args: cobra.MinimumNArgs(1), // server-name
	}

	return cobraCmd, nil
}

func (c *ListCmd) run(cmd *cobra.Command, args []string) error {
	serverName := strings.TrimSpace(args[0])
	if serverName == "" {
		return fmt.Errorf("server-name is required")
	}

	cfg, err := context.LoadExecutionContextConfig(flags.RuntimeFile)
	if err != nil {
		return fmt.Errorf("failed to load execution context config: %w", err)
	}

	server, ok := cfg.Servers[serverName]
	if !ok {
		return fmt.Errorf("server '%s' not found in configuration", serverName)
	}

	fmt.Printf("Environment variables for '%s':\n", serverName)
	if len(server.Env) == 0 {
		fmt.Println("  (No environment variables set)")
		return nil
	}

	keys := make([]string, 0, len(server.Env))
	for k := range server.Env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		fmt.Printf("  %s = %s\n", k, server.Env[k])
	}

	return nil
}
