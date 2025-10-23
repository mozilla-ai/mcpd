package plugins

import (
	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
)

// NewCmd creates the parent plugins command.
func NewCmd(baseCmd *cmd.BaseCmd, opt ...options.CmdOption) (*cobra.Command, error) {
	cobraCmd := &cobra.Command{
		Use:   "plugins",
		Short: "Manage plugin configuration",
		Long: "Manage plugin configuration including plugin entries (authentication, observability, etc.) " +
			"and plugin-level settings (directory path, etc.)",
	}

	// Sub-commands for: mcpd config plugins.
	fns := []func(baseCmd *cmd.BaseCmd, opt ...options.CmdOption) (*cobra.Command, error){
		NewAddCmd,      // add
		NewGetCmd,      // get
		NewListCmd,     // list
		NewMoveCmd,     // move
		NewRemoveCmd,   // remove
		NewSetCmd,      // set
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
