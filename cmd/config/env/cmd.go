package env

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
		Use:   "env",
		Short: "Manages environment variable configuration for MCP servers",
		Long: "Manages environment variable configuration for MCP servers, " +
			"dealing with setting, removing, clearing and listing configuration",
	}

	// Sub-commands for: mcpd config env
	fns := []func(baseCmd *cmd.BaseCmd, opt ...options.CmdOption) (*cobra.Command, error){
		NewSetCmd,    // set
		NewRemoveCmd, // remove
		NewClearCmd,  // clear
		NewListCmd,   // list
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
