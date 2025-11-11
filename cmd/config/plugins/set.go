package plugins

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	internalcmd "github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
)

const (
	flagDir = "dir"
)

// SetCmd represents the command for setting plugin configuration.
// Use NewSetCmd to create instances of SetCmd.
type SetCmd struct {
	*internalcmd.BaseCmd
	cfgLoader config.Loader

	// Plugin-level flags.
	dir string

	// Plugin entry flags.
	category   config.Category
	name       string
	flows      []string
	required   bool
	commitHash string
}

// NewSetCmd creates a new set command for plugin configuration.
func NewSetCmd(baseCmd *internalcmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	opts, err := cmdopts.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &SetCmd{
		BaseCmd:   baseCmd,
		cfgLoader: opts.ConfigLoader,
	}

	cobraCmd := &cobra.Command{
		Use:   "set",
		Short: "Sets top-level config for all plugins, or config for a specific plugin entry",
		Long: `Sets top-level config for all plugins (--dir), or updates config for a specific plugin entry
(--category, --name, --flow). Cannot mix both modes.

Top-level mode sets configuration that applies to all plugins, such as the directory where plugin binaries are located.

Plugin entry mode updates config for an existing plugin. When updating, only the provided flags are changed (partial update).`,
		PreRunE: c.validate,
		RunE:    c.run,
		Args:    cobra.NoArgs,
		Example: `  # Set plugin directory
  mcpd config plugins set --dir=/path/to/plugins

  # Update existing plugin entry (partial update)
  mcpd config plugins set --category=authentication --name=jwt-auth --flow=request --flow=response`,
	}

	// Plugin-level flags.
	cobraCmd.Flags().StringVar(
		&c.dir,
		flagDir,
		"",
		"Directory path for location of plugin binaries (top-level config only)",
	)

	// Plugin entry flags.
	allowedCategories := config.OrderedCategories()
	cobraCmd.Flags().Var(
		&c.category,
		flagCategory,
		fmt.Sprintf("Plugin category (plugin entry only, one of: %s)", allowedCategories.String()),
	)

	cobraCmd.Flags().StringVar(
		&c.name,
		flagName,
		"",
		"Plugin name (plugin entry only)",
	)

	cobraCmd.Flags().StringArrayVar(
		&c.flows,
		flagFlow,
		nil,
		fmt.Sprintf(
			"Flow during which the plugin should execute (%s) (can be repeated, plugin entry only)",
			strings.Join(config.OrderedFlowNames(), ", "),
		),
	)

	cobraCmd.Flags().BoolVar(
		&c.required,
		flagRequired,
		false,
		"Optional, mark plugin as required (plugin entry only)",
	)

	cobraCmd.Flags().StringVar(
		&c.commitHash,
		flagCommitHash,
		"",
		"Optional, commit hash for version validation (plugin entry only)",
	)

	return cobraCmd, nil
}

func (c *SetCmd) validate(cobraCmd *cobra.Command, _ []string) error {
	if err := c.RequireTogether(cobraCmd, flagCategory, flagName); err != nil {
		return err
	}

	isDirSet := cobraCmd.Flags().Changed(flagDir)
	pluginEntryFlags := []string{flagCategory, flagName, flagFlow, flagRequired, flagCommitHash}
	hasPluginEntryFlags := false
	for _, flag := range pluginEntryFlags {
		if cobraCmd.Flags().Changed(flag) {
			hasPluginEntryFlags = true
			break
		}
	}

	// Cannot mix --dir with plugin entry flags.
	if isDirSet && hasPluginEntryFlags {
		return fmt.Errorf(
			"cannot use --%s with plugin entry flags (--%s)",
			flagDir,
			strings.Join(pluginEntryFlags, ", --"),
		)
	}

	// Must provide either --dir OR plugin entry flags.
	if !isDirSet && !hasPluginEntryFlags {
		return fmt.Errorf("provide either --%s or (--%s and --%s)", flagDir, flagCategory, flagName)
	}

	// Validate directory path if set.
	if isDirSet && strings.TrimSpace(c.dir) == "" {
		return fmt.Errorf("plugin directory path cannot be empty")
	}

	// Validate plugin name if set (RequireTogether ensures category and name are both set).
	if hasPluginEntryFlags && strings.TrimSpace(c.name) == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	// Validate flows if provided.
	if cobraCmd.Flags().Changed(flagFlow) {
		flows := config.ParseFlowsDistinct(c.flows)
		if len(flows) == 0 {
			return fmt.Errorf(
				"at least one valid flow is required (%s)",
				strings.Join(config.OrderedFlowNames(), ", "),
			)
		}
	}

	return nil
}

// run loads the config and delegates to the appropriate handler based on which flags were provided.
func (c *SetCmd) run(cmd *cobra.Command, _ []string) error {
	cfg, err := c.LoadConfig(c.cfgLoader)
	if err != nil {
		return err
	}

	// Determine mode and execute.
	if cmd.Flags().Changed(flagDir) {
		return c.setPluginLevelConfig(cmd, cfg)
	}

	return c.setPluginEntry(cmd, cfg)
}

// setPluginLevelConfig sets the top-level plugin directory configuration.
func (c *SetCmd) setPluginLevelConfig(cmd *cobra.Command, cfg *config.Config) error {
	// Initialize plugin config if needed.
	if cfg.Plugins == nil {
		cfg.Plugins = &config.PluginConfig{}
	}

	// Set the directory.
	cfg.Plugins.Dir = c.dir

	// Save config.
	if err := cfg.SaveConfig(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Plugin directory set to: %s\n", c.dir)
	return nil
}

// setPluginEntry creates a new plugin entry or updates an existing one (upsert).
// When creating a new entry, --flow is required. When updating, only the provided flags are changed.
func (c *SetCmd) setPluginEntry(cobraCmd *cobra.Command, cfg *config.Config) error {
	// Check if plugin exists.
	existing, exists := cfg.Plugin(c.category, c.name)

	// Build plugin entry.
	entry := config.PluginEntry{
		Name: c.name,
	}

	// Handle flows (required for create, optional for update).
	switch {
	case cobraCmd.Flags().Changed(flagFlow):
		flows := config.ParseFlowsDistinct(c.flows)
		entry.Flows = slices.Sorted(maps.Keys(flows))
	case exists:
		// Updating: keep existing flows if not provided.
		entry.Flows = existing.Flows
	default:
		return fmt.Errorf("flows are required when creating a new plugin entry")
	}

	// Handle required flag.
	if cobraCmd.Flags().Changed(flagRequired) {
		entry.Required = &c.required
	} else if exists {
		// Updating: keep existing required if not provided.
		entry.Required = existing.Required
	}

	// Handle commit hash.
	if cobraCmd.Flags().Changed(flagCommitHash) {
		if c.commitHash != "" {
			entry.CommitHash = &c.commitHash
		}
	} else if exists {
		// Updating: keep existing commit hash if not provided.
		entry.CommitHash = existing.CommitHash
	}

	// Upsert the plugin entry.
	result, err := cfg.UpsertPlugin(c.category, entry)
	if err != nil {
		return err
	}

	// Provide feedback based on result.
	_, err = fmt.Fprintf(
		cobraCmd.OutOrStdout(),
		"✓ Plugin '%s' configured in category '%s' (operation: %s)\n", c.name, c.category, string(result),
	)

	return err
}
