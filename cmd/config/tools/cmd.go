package tools

import (
	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
)

type Cmd struct {
	*cmd.BaseCmd
}

func NewCmd(baseCmd *cmd.BaseCmd, opt ...options.CmdOption) (*cobra.Command, error) {
	cobraCmd := &cobra.Command{
		Use:   "tools",
		Short: "Manages tools configuration for a registered MCP server",
		Long: "Manages tools configuration for a registered MCP server, " +
			"dealing with setting, removing, and listing tools",
	}

	// Sub-commands for: mcpd config env
	fns := []func(baseCmd *cmd.BaseCmd, opt ...options.CmdOption) (*cobra.Command, error){
		// NewSetCmd,    // set
		NewRemoveCmd, // remove
		// NewListCmd,   // list
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
