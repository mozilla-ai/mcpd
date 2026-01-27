package daemon

import (
	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/internal/cmd"
	"github.com/mozilla-ai/mcpd/internal/cmd/options"
)

func NewCmd(baseCmd *cmd.BaseCmd, opt ...options.CmdOption) (*cobra.Command, error) {
	cobraCmd := &cobra.Command{
		Use:   "daemon",
		Short: "Manages daemon configuration",
		Long:  "Manages daemon configuration in .mcpd.toml including API settings, CORS, timeouts and intervals",
	}

	// Sub-commands for: mcpd config daemon
	fns := []func(baseCmd *cmd.BaseCmd, opt ...options.CmdOption) (*cobra.Command, error){
		NewSetCmd,      // set
		NewGetCmd,      // get
		NewListCmd,     // list
		NewRemoveCmd,   // remove
		NewValidateCmd, // validate
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
