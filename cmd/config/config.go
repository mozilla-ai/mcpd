package config

import (
	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/cmd/config/args"
	"github.com/mozilla-ai/mcpd/cmd/config/daemon"
	"github.com/mozilla-ai/mcpd/cmd/config/env"
	"github.com/mozilla-ai/mcpd/cmd/config/export"
	"github.com/mozilla-ai/mcpd/cmd/config/plugins"
	"github.com/mozilla-ai/mcpd/cmd/config/tools"
	"github.com/mozilla-ai/mcpd/internal/cmd"
	"github.com/mozilla-ai/mcpd/internal/cmd/options"
)

func NewConfigCmd(baseCmd *cmd.BaseCmd, opt ...options.CmdOption) (*cobra.Command, error) {
	cobraCmd := &cobra.Command{
		Use:   "config",
		Short: "Manages configuration for MCP servers",
		Long:  "Manages configuration for MCP servers, dealing with environment variables, command line args and exporting config",
	}

	// Sub-commands for: mcpd config
	fns := []func(baseCmd *cmd.BaseCmd, opt ...options.CmdOption) (*cobra.Command, error){
		args.NewCmd,    // args
		daemon.NewCmd,  // daemon
		env.NewCmd,     // env
		plugins.NewCmd, // plugins
		tools.NewCmd,   // tools
		export.NewCmd,  // export
	}

	for _, fn := range fns {
		tempCmd, err := fn(baseCmd, opt...)
		if err != nil {
			return nil, err
		}
		cobraCmd.AddCommand(tempCmd)
	}

	return cobraCmd, nil
}
