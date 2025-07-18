package env

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
	"github.com/mozilla-ai/mcpd/v2/internal/context"
	"github.com/mozilla-ai/mcpd/v2/internal/flags"
)

type ListCmd struct {
	*cmd.BaseCmd
	cfgLoader config.Loader
	ctxLoader context.Loader
}

func NewListCmd(baseCmd *cmd.BaseCmd, opt ...options.CmdOption) (*cobra.Command, error) {
	opts, err := options.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &ListCmd{
		BaseCmd:   baseCmd,
		cfgLoader: opts.ConfigLoader,
		ctxLoader: opts.ContextLoader,
	}

	cobraCmd := &cobra.Command{
		Use:   "list <server-name>",
		Short: "Lists configured environment variables for a specific MCP server",
		Long: "Lists configured environment variables for a specific MCP server, using the " +
			"runtime context configuration file (e.g. `~/.config/mcpd/secrets.dev.toml`)",
		RunE: c.run,
		Args: cobra.MinimumNArgs(1), // server-name
	}

	return cobraCmd, nil
}

func (c *ListCmd) run(_ *cobra.Command, args []string) error {
	serverName := strings.TrimSpace(args[0])
	if serverName == "" {
		return fmt.Errorf("server-name is required")
	}

	cfg, err := c.ctxLoader.Load(flags.RuntimeFile)
	if err != nil {
		return fmt.Errorf("failed to load execution context config: %w", err)
	}

	server, ok := cfg.Get(serverName)
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
