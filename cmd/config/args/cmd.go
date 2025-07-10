package args

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
		Use:   "args",
		Short: "Manages MCP server command line args configuration",
		Long:  "Manages MCP server command line args configuration, dealing with setting, removing, clearing and listing configuration",
	}

	// Sub-commands for: mcpd config args
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
