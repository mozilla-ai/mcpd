package plugins

import (
	"fmt"

	"github.com/spf13/cobra"

	internalcmd "github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd/output"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
	"github.com/mozilla-ai/mcpd/v2/internal/printer"
)

const (
	flagCategory = "category"
	flagName     = "name"
)

// GetCmd represents the get command.
// NOTE: Use NewGetCmd to create a GetCmd.
type GetCmd struct {
	*internalcmd.BaseCmd

	// cfgLoader is used to load the configuration.
	cfgLoader config.Loader

	// pluginConfigPrinter is used to output top-level plugin configuration.
	pluginConfigPrinter output.Printer[printer.PluginConfigResult]

	// pluginEntryPrinter is used to output specific plugin entries.
	pluginEntryPrinter output.Printer[printer.PluginEntryResult]

	// format stores the format flag when specified.
	format internalcmd.OutputFormat

	// category is the (optional) category name to look in for the plugin.
	category config.Category

	// name is the (optional) name of the plugin to return config for.
	name string
}

// NewGetCmd creates the get command for plugins.
func NewGetCmd(baseCmd *internalcmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	opts, err := cmdopts.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &GetCmd{
		BaseCmd:             baseCmd,
		cfgLoader:           opts.ConfigLoader,
		pluginConfigPrinter: &printer.PluginConfigPrinter{},
		pluginEntryPrinter:  &printer.PluginEntryPrinter{},
		format:              internalcmd.FormatText, // Default to plain text
	}

	cobraCmd := &cobra.Command{
		Use:   "get",
		Short: "Get top level configuration for the plugin subsystem or for a specific plugin entry in a category",
		Long: `Get top level configuration for the plugin subsystem or for a specific plugin entry in a category.

When called without flags, shows plugin-level configuration (e.g. plugin directory).
When called with --category and --name, shows the specific plugin entry.
Both --category and --name must be provided together.`,
		Example: `  # Get plugin-level configuration
  mcpd config plugins get

  # Get specific plugin entry
  mcpd config plugins get --category authentication --name jwt-auth`,
		RunE: c.run,
		Args: cobra.NoArgs,
	}

	allowedCategories := config.OrderedCategories()
	cobraCmd.Flags().Var(
		&c.category,
		"category",
		fmt.Sprintf("Specify the category (one of: %s)", allowedCategories.String()),
	)

	cobraCmd.Flags().StringVar(
		&c.name,
		flagName,
		"",
		"Plugin name",
	)

	allowedOutputFormats := internalcmd.AllowedOutputFormats()
	cobraCmd.Flags().Var(
		&c.format,
		"format",
		fmt.Sprintf("Specify the output format (one of: %s)", allowedOutputFormats.String()),
	)

	return cobraCmd, nil
}

func (c *GetCmd) run(cmd *cobra.Command, _ []string) error {
	// Validate optional flags.
	if err := c.RequireTogether(cmd, flagCategory, flagName); err != nil {
		return err
	}

	cfg, err := c.LoadConfig(c.cfgLoader)
	if err != nil {
		return err
	}

	// Show top-level plugin config settings.
	if !cmd.Flags().Changed(flagCategory) {
		var dir string
		if cfg.Plugins != nil {
			dir = cfg.Plugins.Dir
		}

		result := printer.PluginConfigResult{
			Dir: dir,
		}

		handler, err := internalcmd.FormatHandler(cmd.OutOrStdout(), c.format, c.pluginConfigPrinter)
		if err != nil {
			return err
		}

		return handler.HandleResult(result)
	}

	// Show a single plugin item for category/name if we can find it.
	if entry, found := cfg.Plugin(c.category, c.name); found {
		result := printer.PluginEntryResult{
			PluginEntry: entry,
			Category:    c.category,
		}

		handler, err := internalcmd.FormatHandler(cmd.OutOrStdout(), c.format, c.pluginEntryPrinter)
		if err != nil {
			return err
		}

		return handler.HandleResult(result)
	}

	return fmt.Errorf("plugin '%s' not found in category '%s'", c.name, c.category)
}
