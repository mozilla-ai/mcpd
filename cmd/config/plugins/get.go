package plugins

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
)

// NewGetCmd creates the get command for plugins.
// TODO: Implement in a future PR.
func NewGetCmd(baseCmd *cmd.BaseCmd, opt ...options.CmdOption) (*cobra.Command, error) {
	cobraCmd := &cobra.Command{
		Use:   "get",
		Short: "Get plugin-level config or specific plugin entry",
		Long:  "Get plugin-level configuration (when no flags provided) or specific plugin entry (when --category and --name provided)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("not yet implemented")
		},
	}

	return cobraCmd, nil
}
