package plugins

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
)

// NewMoveCmd creates the move command for plugins.
// TODO: Implement in a future PR.
func NewMoveCmd(baseCmd *cmd.BaseCmd, opt ...options.CmdOption) (*cobra.Command, error) {
	cobraCmd := &cobra.Command{
		Use:   "move",
		Short: "Reorder a plugin entry within its category",
		Long:  "Reorder a plugin entry within its category. Plugin execution order matters.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented")
		},
	}

	return cobraCmd, nil
}
