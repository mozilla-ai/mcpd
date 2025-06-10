package server

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd-cli/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd-cli/v2/internal/config"
	"github.com/mozilla-ai/mcpd-cli/v2/internal/flags"
)

// RemoveCmd should be used to represent the 'remove' command.
type RemoveCmd struct {
	*cmd.BaseCmd
}

// NewRemoveCmd creates a newly configured (Cobra) command.
func NewRemoveCmd(baseCmd *cmd.BaseCmd) *cobra.Command {
	c := &RemoveCmd{
		BaseCmd: baseCmd,
	}

	cobraCommand := &cobra.Command{
		Use:   "remove <server-name>",
		Short: "Removes an MCP server dependency from the project.",
		Long:  c.longDescription(),
		RunE:  c.run,
	}

	return cobraCommand
}

// longDescription returns the long version of the command description.
func (c *RemoveCmd) longDescription() string {
	return `Removes an MCP server dependency from the project config file.
Specify the server name to remove it.`
}

// run is configured (via NewRemoveCmd) to be called by the Cobra framework when the command is executed.
// It may return an error (or nil, when there is no error).
func (c *RemoveCmd) run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("server name is required and cannot be empty")
	}

	logger := c.Logger()

	name := strings.TrimSpace(args[0])
	if name == "" {
		return fmt.Errorf("server name cannot be empty")
	}

	cfg, err := config.NewConfig(flags.ConfigFile)
	if err != nil {
		return err
	}

	err = cfg.RemoveServer(name)
	if err != nil {
		return err
	}

	logger.Debug("Server removed", "name", name)
	fmt.Fprintf(cmd.OutOrStdout(), "✓ Removed server '%s'\n", name)

	return nil
}
