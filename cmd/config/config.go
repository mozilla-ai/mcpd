package config

import (
	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/cmd/config/args"
	"github.com/mozilla-ai/mcpd/v2/cmd/config/env"
	"github.com/mozilla-ai/mcpd/v2/cmd/config/export"
	"github.com/mozilla-ai/mcpd/v2/cmd/config/tools"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
)

func NewConfigCmd(baseCmd *cmd.BaseCmd, opt ...options.CmdOption) (*cobra.Command, error) {
	cobraCmd := &cobra.Command{
		Use:   "config",
		Short: "Manages configuration for MCP servers",
		Long:  "Manages configuration for MCP servers, dealing with environment variables, command line args and exporting config",
	}

	// Sub-commands for: mcpd config
	fns := []func(baseCmd *cmd.BaseCmd, opt ...options.CmdOption) (*cobra.Command, error){
		args.NewCmd,   // args
		env.NewCmd,    // env
		tools.NewCmd,  // tools
		export.NewCmd, // export
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
