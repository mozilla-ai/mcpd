package plugins

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
)

// NewAddCmd creates the add command for plugins.
// TODO: Implement in a future PR.
func NewAddCmd(baseCmd *cmd.BaseCmd, opt ...options.CmdOption) (*cobra.Command, error) {
	cobraCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new plugin entry to a category",
		Long:  "Add a new plugin entry to a category. The configuration is saved automatically.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented")
		},
	}

	return cobraCmd, nil
}
