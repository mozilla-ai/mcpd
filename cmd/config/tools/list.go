package tools

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	internalcmd "github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd/output"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
	"github.com/mozilla-ai/mcpd/v2/internal/flags"
	"github.com/mozilla-ai/mcpd/v2/internal/printer"
)

type ListCmd struct {
	*internalcmd.BaseCmd
	cfgLoader    config.Loader
	Format       internalcmd.OutputFormat
	toolsPrinter output.Printer[printer.ToolsListResult]
}

func NewListCmd(baseCmd *internalcmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	opts, err := cmdopts.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &ListCmd{
		BaseCmd:      baseCmd,
		cfgLoader:    opts.ConfigLoader,
		Format:       internalcmd.FormatText, // Default to plain text
		toolsPrinter: &printer.ToolsListPrinter{},
	}

	cobraCmd := &cobra.Command{
		Use:   "list <server-name>",
		Short: "Lists the configured tools for a specific MCP server",
		Long:  "Lists the configured tools for a specific MCP server from the .mcpd.toml configuration file",
		RunE:  c.run,
		Args:  cobra.ExactArgs(1),
	}

	// Add format flag
	allowed := internalcmd.AllowedOutputFormats()
	cobraCmd.Flags().Var(
		&c.Format,
		"format",
		fmt.Sprintf("Specify the output format (one of: %s)", allowed.String()),
	)

	return cobraCmd, nil
}

func (c *ListCmd) run(cmd *cobra.Command, args []string) error {
	handler, err := internalcmd.FormatHandler(cmd.OutOrStdout(), c.Format, c.toolsPrinter)
	if err != nil {
		return err
	}

	serverName := strings.TrimSpace(args[0])
	if serverName == "" {
		return handler.HandleError(fmt.Errorf("server-name is required"))
	}

	cfg, err := c.cfgLoader.Load(flags.ConfigFile)
	if err != nil {
		return handler.HandleError(err)
	}

	for _, srv := range cfg.ListServers() {
		if srv.Name != serverName {
			continue
		}

		// Sort tools alphabetically for consistent output.
		tools := make([]string, len(srv.Tools))
		copy(tools, srv.Tools)
		sort.Strings(tools)

		result := printer.ToolsListResult{
			Server: serverName,
			Tools:  tools,
			Count:  len(tools),
		}

		return handler.HandleResult(result)
	}

	return handler.HandleError(fmt.Errorf("server '%s' not found in configuration", serverName))
}
