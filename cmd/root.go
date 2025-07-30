package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/cmd/config"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/flags"
)

type RootCmd struct {
	*cmd.BaseCmd
}

// Global variable to hold the root command instance
var rootCmdInstance *RootCmd

func Execute() error {
	// Create the root command instance
	rootCmdInstance = &RootCmd{
		BaseCmd: &cmd.BaseCmd{},
	}

	// Create cobra command.
	rootCmd, err := NewRootCmd(rootCmdInstance)
	if err != nil {
		return fmt.Errorf("could not create root command: %w", err)
	}

	if err := rootCmd.Execute(); err != nil {
		return err
	}

	return nil
}

func NewRootCmd(c *RootCmd) (*cobra.Command, error) {
	rootCmd := &cobra.Command{
		Use:   "mcpd",
		Short: "`mcpd` CLI is the primary interface for developers to interact with, and configure `mcpd`",
		Long: "The `mcpd` CLI is the primary interface for developers to interact with `mcpd` " +
			"define their agent projects, and manage MCP server dependencies",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Manually set the version template to prevent duplication in output.
	rootCmd.SetVersionTemplate("{{.Version}}\n")
	rootCmd.Version = cmd.Version()

	// Configure app specific global flags that will appear on sub-commands.
	if err := flags.InitFlags(rootCmd.PersistentFlags()); err != nil {
		return nil, err
	}

	// Create 'Core' commands for top-level application commands.
	rootCmd.AddGroup(&cobra.Group{
		ID:    "core",
		Title: "Core Commands:",
	})

	// Create top-level commands and add to 'Core' commands group.
	fns := []func(baseCmd *cmd.BaseCmd, opt ...options.CmdOption) (*cobra.Command, error){
		NewInitCmd,
		NewSearchCmd,
		NewAddCmd,
		NewRemoveCmd,
		NewDaemonCmd,
		config.NewConfigCmd,
	}

	for _, fn := range fns {
		tempCmd, err := fn(c.BaseCmd, options.WithRegistryBuilder(c.BaseCmd))
		if err != nil {
			return nil, err
		}

		// Associate the command with the core group.
		tempCmd.GroupID = "core"

		rootCmd.AddCommand(tempCmd)
	}

	// Assign built-in commands (e.g. help, completion) to the 'system' group.
	rootCmd.AddGroup(&cobra.Group{
		ID:    "system",
		Title: "System Commands:",
	})
	rootCmd.SetHelpCommandGroupID("system")
	rootCmd.SetCompletionCommandGroupID("system")

	// Hide the --help flag on all commands.
	hideHelpFlagsRecursively(rootCmd)

	return rootCmd, nil
}

func hideHelpFlagsRecursively(cmd *cobra.Command) {
	// Ensure the command has the help flag initialized.
	cmd.InitDefaultHelpFlag()

	if f := cmd.Flags().Lookup("help"); f != nil {
		f.Hidden = true
	}

	// Recurse into children (depth first).
	for _, subCmd := range cmd.Commands() {
		hideHelpFlagsRecursively(subCmd)
	}
}
