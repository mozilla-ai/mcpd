package plugins

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
)

// NewSetCmd creates the set command for plugins.
// TODO: Implement in a future PR.
func NewSetCmd(baseCmd *cmd.BaseCmd, opt ...options.CmdOption) (*cobra.Command, error) {
	cobraCmd := &cobra.Command{
		Use:   "set",
		Short: "Set plugin-level config or plugin entry",
		Long:  "Set plugin-level configuration (--dir) or create/update a plugin entry (--category, --name, --flows)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented")
		},
	}

	return cobraCmd, nil
}
