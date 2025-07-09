package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/cmd/config"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/flags"
	"github.com/mozilla-ai/mcpd/v2/internal/printer"
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
		Use:           "mcpd <command> [sub-command] [args]",
		Short:         "'mcpd' CLI is the primary interface for developers to interact with mcpd.",
		Long:          c.longDescription(),
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       cmd.Version(),
	}

	// Global flags
	if err := flags.InitFlags(rootCmd.PersistentFlags()); err != nil {
		return nil, err
	}

	// Add top-level commands
	fns := []func(baseCmd *cmd.BaseCmd, opt ...options.CmdOption) (*cobra.Command, error){
		NewInitCmd,
		NewSearchCmd,
		NewAddCmd,
		NewRemoveCmd,
		NewDaemonCmd,
		config.NewConfigCmd,
	}

	for _, fn := range fns {
		p, err := printer.NewPrinter(rootCmd.OutOrStdout())
		if err != nil {
			return nil, err
		}

		opts := []options.CmdOption{
			options.WithPrinter(p),
			options.WithRegistryBuilder(c.BaseCmd),
		}

		tempCmd, err := fn(c.BaseCmd, opts...)
		if err != nil {
			return nil, err
		}
		rootCmd.AddCommand(tempCmd)
	}

	return rootCmd, nil
}

func (c *RootCmd) longDescription() string {
	return `The 'mcpd' CLI is the primary interface for developers to interact with the
mcpd Control Plane, define their agent projects, and manage MCP server dependencies.`
}
