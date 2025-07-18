package args

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/context"
	"github.com/mozilla-ai/mcpd/v2/internal/flags"
)

type ListCmd struct {
	*cmd.BaseCmd
	ctxLoader context.Loader
}

func NewListCmd(baseCmd *cmd.BaseCmd, opt ...options.CmdOption) (*cobra.Command, error) {
	opts, err := options.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &ListCmd{
		BaseCmd:   baseCmd,
		ctxLoader: opts.ContextLoader,
	}

	cobraCmd := &cobra.Command{
		Use:   "list <server-name>",
		Short: "Lists the configured command line arguments for a specific MCP server",
		Long: "Lists the configured command line arguments for a specific MCP server, " +
			"using the runtime context configuration file (e.g. `~/.config/mcpd/secrets.dev.toml`)",
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

	fmt.Printf("Arguments for '%s':\n", serverName)
	if len(server.Args) == 0 {
		fmt.Println("  (No arguments set)")
		return nil
	}

	for _, arg := range server.Args {
		fmt.Printf("  %s\n", arg)
	}

	return nil
}
