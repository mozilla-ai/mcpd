package plugins

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
)

// NewRemoveCmd creates the remove command for plugins.
// TODO: Implement in a future PR.
func NewRemoveCmd(baseCmd *cmd.BaseCmd, opt ...options.CmdOption) (*cobra.Command, error) {
	cobraCmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove a plugin entry from a category",
		Long:  "Remove a plugin entry from a category. The configuration is saved automatically.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented")
		},
	}

	return cobraCmd, nil
}
