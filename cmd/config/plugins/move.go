package plugins

import (
	"fmt"

	"github.com/spf13/cobra"

	internalcmd "github.com/mozilla-ai/mcpd/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/internal/config"
)

const (
	flagToCategory = "to-category"
	flagBefore     = "before"
	flagAfter      = "after"
	flagPosition   = "position"
	flagForce      = "force"
)

// MoveCmd represents the command for moving/reordering a plugin entry.
// Use NewMoveCmd to create instances of MoveCmd.
type MoveCmd struct {
	*internalcmd.BaseCmd

	// after positions the plugin after this target plugin.
	after string

	// before positions the plugin before this target plugin.
	before string

	// category is the current category of the plugin to move.
	category config.Category

	// cfgLoader loads the configuration file.
	cfgLoader config.Loader

	// force overwrites an existing plugin in the target category.
	force bool

	// name is the name of the plugin to move.
	name string

	// position is the absolute position (1-based, -1 for end).
	position int

	// toCategory is the optional destination category.
	toCategory config.Category
}

// NewMoveCmd creates a new move command for moving plugin entries.
func NewMoveCmd(baseCmd *internalcmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	opts, err := cmdopts.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &MoveCmd{
		BaseCmd:   baseCmd,
		cfgLoader: opts.ConfigLoader,
		position:  -1, // Default: not set.
	}

	cobraCmd := &cobra.Command{
		Use:   "move",
		Short: "Move a plugin between categories or reorder within a category",
		Long: `Move a plugin entry between categories and/or reorder within a category.
Plugin execution order matters, so use this command to control both
categorization and position in the execution pipeline.`,
		PreRunE: c.validate,
		RunE:    c.run,
		Args:    cobra.NoArgs,
		Example: `  # Move to different category
  mcpd config plugins move --category=authentication --name=jwt-auth --to-category=audit

  # Move to different category at specific position
  mcpd config plugins move --category=authentication --name=jwt-auth --to-category=audit --position=1

  # Move before another plugin (same category)
  mcpd config plugins move --category=authentication --name=jwt-auth --before=oauth

  # Move after another plugin (same category)
  mcpd config plugins move --category=authentication --name=jwt-auth --after=api-key

  # Move to specific position (same category)
  mcpd config plugins move --category=authentication --name=jwt-auth --position=1`,
	}

	allowedCategories := config.OrderedCategories()
	cobraCmd.Flags().Var(
		&c.category,
		flagCategory,
		fmt.Sprintf("Current category of the plugin to move (one of: %s)", allowedCategories.String()),
	)
	_ = cobraCmd.MarkFlagRequired(flagCategory)

	cobraCmd.Flags().StringVar(
		&c.name,
		flagName,
		"",
		"Name of the plugin to move",
	)
	_ = cobraCmd.MarkFlagRequired(flagName)

	cobraCmd.Flags().Var(
		&c.toCategory,
		flagToCategory,
		fmt.Sprintf("Optional, move plugin to this category (one of: %s)", allowedCategories.String()),
	)

	cobraCmd.Flags().StringVar(
		&c.before,
		flagBefore,
		"",
		"Optional, position the named plugin before this plugin",
	)

	cobraCmd.Flags().StringVar(
		&c.after,
		flagAfter,
		"",
		"Optional, position the named plugin after this plugin",
	)

	cobraCmd.Flags().IntVar(
		&c.position,
		flagPosition,
		-1,
		"Optional, an absolute position (order) for the plugin (use: -1 to move to the end)",
	)

	cobraCmd.Flags().BoolVar(
		&c.force,
		flagForce,
		false,
		"Optional, overwrite existing plugin in target category if it exists",
	)

	// Can't specify a specific position and a relative one at the same time.
	cobraCmd.MarkFlagsMutuallyExclusive(flagBefore, flagPosition)
	cobraCmd.MarkFlagsMutuallyExclusive(flagAfter, flagPosition)
	// Can't specify being before AND after a named plugin.
	cobraCmd.MarkFlagsMutuallyExclusive(flagAfter, flagBefore)

	return cobraCmd, nil
}

// buildOptions constructs MoveOption slice from the provided command flags.
func (c *MoveCmd) buildOptions(cobraCmd *cobra.Command) []config.MoveOption {
	var opts []config.MoveOption

	if cobraCmd.Flags().Changed(flagToCategory) {
		opts = append(opts, config.WithToCategory(c.toCategory))
	}
	if cobraCmd.Flags().Changed(flagBefore) {
		opts = append(opts, config.WithBefore(c.before))
	}
	if cobraCmd.Flags().Changed(flagAfter) {
		opts = append(opts, config.WithAfter(c.after))
	}
	if cobraCmd.Flags().Changed(flagPosition) {
		opts = append(opts, config.WithPosition(c.position))
	}
	if cobraCmd.Flags().Changed(flagForce) {
		opts = append(opts, config.WithForce(c.force))
	}

	return opts
}

// printOrder displays the current plugin order in the target category.
func (c *MoveCmd) printOrder(cobraCmd *cobra.Command, cfg *config.Config) {
	// Determine target category: toCategory if set, otherwise original category.
	targetCategory := c.category
	if c.toCategory != "" {
		targetCategory = c.toCategory
	}

	plugins := cfg.Plugins.ListPlugins(targetCategory)
	if len(plugins) == 0 {
		return
	}

	_, _ = fmt.Fprintf(cobraCmd.OutOrStdout(), "Order in '%s':\n", targetCategory)
	for i, plugin := range plugins {
		_, _ = fmt.Fprintf(cobraCmd.OutOrStdout(), "  %d. %s\n", i+1, plugin.Name)
	}
}

// run executes the move operation and prints the result.
func (c *MoveCmd) run(cobraCmd *cobra.Command, _ []string) error {
	cfg, err := c.LoadConfig(c.cfgLoader)
	if err != nil {
		return err
	}

	result, err := cfg.MovePlugin(c.category, c.name, c.buildOptions(cobraCmd)...)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(cobraCmd.OutOrStdout(), "✓ Plugin '%s' moved (operation: %s)\n\n", c.name, result)
	c.printOrder(cobraCmd, cfg)

	return nil
}

// validate ensures the flag combination is valid.
// NOTE: Mutual exclusivity flags should be handled by Cobra parsing args.
func (c *MoveCmd) validate(cobraCmd *cobra.Command, _ []string) error {
	toCategorySet := cobraCmd.Flags().Changed(flagToCategory)
	hasPosition := cobraCmd.Flags().Changed(flagPosition)
	if hasPosition && c.position != -1 && c.position < 1 {
		return fmt.Errorf("invalid '%s' flag value (must be a positive integer or -1 for end)", flagPosition)
	}

	hasPositioning := cobraCmd.Flags().Changed(flagBefore) || cobraCmd.Flags().Changed(flagAfter) || hasPosition

	// Must specify at least one operation.
	if !toCategorySet && !hasPositioning {
		return fmt.Errorf("one of --to-category, --before, --after, or --position must be specified")
	}

	// Cannot move to the same category.
	if toCategorySet && c.category == c.toCategory {
		return fmt.Errorf(
			"plugin is already in category '%s', "+
				"to reorder within the same category use --before, --after, or --position without --to-category",
			c.category,
		)
	}

	return nil
}
