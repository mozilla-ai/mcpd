package volumes

import (
	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/internal/cmd/options"
)

// NewCmd creates a new volumes command with its sub-commands.
func NewCmd(baseCmd *cmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	cobraCmd := &cobra.Command{
		Use:   "volumes",
		Short: "Manages volume configuration for MCP servers",
		Long: "Manages Docker volume configuration for MCP servers, " +
			"dealing with setting, removing, clearing and listing volume mappings.",
	}

	fns := []func(baseCmd *cmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error){
		NewRemoveCmd,
		NewSetCmd,
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
