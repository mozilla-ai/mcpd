package plugins

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	internalcmd "github.com/mozilla-ai/mcpd/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/internal/config"
)

const (
	flagFlow       = "flow"
	flagRequired   = "required"
	flagCommitHash = "commit-hash"
)

// AddCmd represents the command for adding a new plugin entry.
// NOTE: Use NewAddCmd to create instances of AddCmd.
type AddCmd struct {
	*internalcmd.BaseCmd

	// cfgLoader is used to load the configuration.
	cfgLoader config.Loader

	// category is the category to add the plugin to.
	category config.Category

	// flows is the list of flows.
	flows []string

	// required indicates if the plugin is required.
	required bool

	// commitHash is the optional commit hash for version validation.
	commitHash string
}

// NewAddCmd creates a new add command for plugin entries.
func NewAddCmd(baseCmd *internalcmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	opts, err := cmdopts.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &AddCmd{
		BaseCmd:   baseCmd,
		cfgLoader: opts.ConfigLoader,
	}

	cobraCmd := &cobra.Command{
		Use:   "add <plugin-name>",
		Short: "Add a new plugin entry to a category",
		Long: `Add a new plugin entry to a category. The configuration is saved automatically.

The plugin name must exactly match the name of the plugin binary file.

This command creates new plugin entries only. If a plugin with the same name already exists
in the category, the command fails with an error. To update an existing plugin, use the 'set' command.`,
		Example: `  # Add new plugin with all fields
  mcpd config plugins add jwt-auth --category=authentication --flow=request --required

  # Add plugin with multiple flows
  mcpd config plugins add metrics --category=observability --flow=request --flow=response --commit-hash=abc123

  # Add without required flag (defaults to false)
  mcpd config plugins add rbac --category=authorization --flow=response`,
		RunE: c.run,
		Args: cobra.ExactArgs(1), // plugin-name
	}

	allowedCategories := config.OrderedCategories()
	cobraCmd.Flags().Var(
		&c.category,
		flagCategory,
		fmt.Sprintf("Specify the category (one of: %s)", allowedCategories.String()),
	)
	_ = cobraCmd.MarkFlagRequired(flagCategory)

	cobraCmd.Flags().StringArrayVar(
		&c.flows,
		flagFlow,
		nil,
		fmt.Sprintf(
			"Flow during which, the plugin should execute (%s) (can be repeated)",
			strings.Join(config.OrderedFlowNames(), ", "),
		),
	)
	_ = cobraCmd.MarkFlagRequired(flagFlow)

	cobraCmd.Flags().BoolVar(
		&c.required,
		flagRequired,
		false,
		"Optional, mark plugin as required",
	)

	cobraCmd.Flags().StringVar(
		&c.commitHash,
		flagCommitHash,
		"",
		"Optional, commit hash for runtime version validation",
	)

	return cobraCmd, nil
}

func (c *AddCmd) run(cmd *cobra.Command, args []string) error {
	pluginName := strings.TrimSpace(args[0])
	if pluginName == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	cfg, err := c.LoadConfig(c.cfgLoader)
	if err != nil {
		return err
	}

	if _, exists := cfg.Plugin(c.category, pluginName); exists {
		return fmt.Errorf(
			"plugin '%s' already exists in category '%s'\n\n"+
				"To update an existing plugin, use: mcpd config plugins set %s --category=%s [flags]",
			pluginName,
			c.category,
			pluginName,
			c.category,
		)
	}

	flows := config.ParseFlowsDistinct(c.flows)
	if len(flows) == 0 {
		return fmt.Errorf(
			"at least one valid flow is required (%s)",
			strings.Join(config.OrderedFlowNames(), ", "),
		)
	}

	entry := config.PluginEntry{
		Name:  pluginName,
		Flows: slices.Sorted(maps.Keys(flows)),
	}

	// Set optional fields only if they were provided.
	if cmd.Flags().Changed(flagRequired) {
		entry.Required = &c.required
	}

	if c.commitHash != "" {
		entry.CommitHash = &c.commitHash
	}

	if _, err := cfg.UpsertPlugin(c.category, entry); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(
		cmd.OutOrStdout(),
		"✓ Plugin '%s' added to category '%s'\n",
		pluginName,
		c.category,
	)

	return nil
}
