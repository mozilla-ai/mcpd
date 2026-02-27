package volumes

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/internal/context"
	"github.com/mozilla-ai/mcpd/internal/flags"
)

// listCmd handles listing volume mappings for an MCP server.
type listCmd struct {
	*cmd.BaseCmd
	ctxLoader context.Loader
}

// NewListCmd creates a new list command for volume configuration.
func NewListCmd(baseCmd *cmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	opts, err := cmdopts.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &listCmd{
		BaseCmd:   baseCmd,
		ctxLoader: opts.ContextLoader,
	}

	cobraCmd := &cobra.Command{
		Use:   "list <server-name>",
		Short: "List configured volume mappings for a server",
		Long: "List configured volume mappings for an MCP server from the " +
			"runtime context configuration file (e.g. " + flags.RuntimeFile + ").",
		RunE: c.run,
		Args: cobra.ExactArgs(1),
	}

	return cobraCmd, nil
}

// run executes the list command, displaying volume mappings for the given server.
func (c *listCmd) run(cobraCmd *cobra.Command, args []string) error {
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

	out := cobraCmd.OutOrStdout()

	if _, err := fmt.Fprintf(out, "Volumes for '%s':\n", serverName); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	if len(server.Volumes) == 0 {
		if _, err := fmt.Fprintln(out, "  (No volumes set)"); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
		return nil
	}

	// Sort keys for deterministic output.
	keys := slices.Collect(maps.Keys(server.Volumes))
	slices.Sort(keys)

	for _, k := range keys {
		if _, err := fmt.Fprintf(out, "  %s = %s\n", k, server.Volumes[k]); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
	}

	return nil
}
