package plugins

import (
	"fmt"

	"github.com/spf13/cobra"

	internalcmd "github.com/mozilla-ai/mcpd/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/internal/config"
)

// RemoveCmd represents the command for removing a plugin entry.
// Use NewRemoveCmd to create instances of RemoveCmd.
type RemoveCmd struct {
	*internalcmd.BaseCmd

	// cfgLoader is used to load the configuration.
	cfgLoader config.Loader

	// category is the category to remove the plugin from.
	category config.Category

	// name is the name of the plugin to remove.
	name string
}

// NewRemoveCmd creates a new remove command for plugin entries.
func NewRemoveCmd(baseCmd *internalcmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	opts, err := cmdopts.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &RemoveCmd{
		BaseCmd:   baseCmd,
		cfgLoader: opts.ConfigLoader,
	}

	cobraCmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove a plugin entry from a category",
		Long:  "Remove a plugin entry from a category. The configuration is saved automatically.",
		RunE:  c.run,
		Args:  cobra.NoArgs,
		Example: `  # Remove a plugin entry
  mcpd config plugins remove --category=authentication --name=jwt-auth

  # Remove from observability category
  mcpd config plugins remove --category=observability --name=metrics`,
	}

	allowedCategories := config.OrderedCategories()
	cobraCmd.Flags().Var(
		&c.category,
		flagCategory,
		fmt.Sprintf("Category of the plugin to remove (one of: %s)", allowedCategories.String()),
	)
	_ = cobraCmd.MarkFlagRequired(flagCategory)

	cobraCmd.Flags().StringVar(
		&c.name,
		flagName,
		"",
		"Name of the plugin to remove",
	)
	_ = cobraCmd.MarkFlagRequired(flagName)

	return cobraCmd, nil
}

// run executes the remove command.
func (c *RemoveCmd) run(cmd *cobra.Command, _ []string) error {
	cfg, err := c.LoadConfig(c.cfgLoader)
	if err != nil {
		return err
	}

	result, err := cfg.DeletePlugin(c.category, c.name)
	if err != nil {
		return fmt.Errorf("error removing plugin '%s' from category '%s': %w", c.name, c.category, err)
	}

	_, _ = fmt.Fprintf(
		cmd.OutOrStdout(),
		"✓ Plugin '%s' removed from category '%s' (operation: %s)\n",
		c.name,
		c.category,
		string(result),
	)

	return nil
}
