package plugins

import (
	"fmt"

	"github.com/spf13/cobra"

	internalcmd "github.com/mozilla-ai/mcpd/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/internal/cmd/output"
	"github.com/mozilla-ai/mcpd/internal/config"
	"github.com/mozilla-ai/mcpd/internal/printer"
)

// ListCmd represents the command for listing configured plugins.
// Use NewListCmd to create instances of ListCmd.
type ListCmd struct {
	*internalcmd.BaseCmd

	// cfgLoader is used to load the configuration.
	cfgLoader config.Loader

	// printer is used to output configured plugins.
	printer output.Printer[printer.PluginListResult]

	// format stores the format flag when specified.
	format internalcmd.OutputFormat

	// category stores the category flag when specified.
	category config.Category
}

// NewListCmd creates a new list command for displaying configured plugins.
func NewListCmd(baseCmd *internalcmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	opts, err := cmdopts.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &ListCmd{
		BaseCmd:   baseCmd,
		cfgLoader: opts.ConfigLoader,
		printer:   &printer.PluginListPrinter{},
		format:    internalcmd.FormatText, // Default to plain text
	}

	cobraCmd := &cobra.Command{
		Use:   "list",
		Short: "List configured plugin entries",
		Long:  "List configured plugin entries in a specific category or across all categories",
		Example: `  # List plugins in authentication category
  mcpd config plugins list --category=authentication

  # List all plugins across all categories
  mcpd config plugins list

  # List with JSON output
  mcpd config plugins list --category=observability --format=json`,
		RunE: c.run,
		Args: cobra.NoArgs,
	}

	allowedOutputFormats := internalcmd.AllowedOutputFormats()
	cobraCmd.Flags().Var(
		&c.format,
		"format",
		fmt.Sprintf("Specify the output format (one of: %s)", allowedOutputFormats.String()),
	)

	allowedCategories := config.OrderedCategories()
	cobraCmd.Flags().Var(
		&c.category,
		"category",
		fmt.Sprintf("Specify the category (one of: %s)", allowedCategories.String()),
	)

	return cobraCmd, nil
}

func (c *ListCmd) run(cmd *cobra.Command, _ []string) error {
	handler, err := internalcmd.FormatHandler(cmd.OutOrStdout(), c.format, c.printer)
	if err != nil {
		return err
	}

	cfg, err := c.LoadConfig(c.cfgLoader)
	if err != nil {
		return handler.HandleError(err)
	}

	if cfg.Plugins == nil {
		result := printer.NewPluginListResult(nil)
		return handler.HandleResult(result)
	}

	var categories map[config.Category][]config.PluginEntry

	if c.category != "" {
		// List single category.
		categories = map[config.Category][]config.PluginEntry{
			c.category: cfg.Plugins.ListPlugins(c.category),
		}
	} else {
		// List all categories.
		categories = cfg.Plugins.AllCategories()
	}

	result := printer.NewPluginListResult(categories)

	return handler.HandleResult(result)
}
