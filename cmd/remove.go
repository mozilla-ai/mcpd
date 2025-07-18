package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
	"github.com/mozilla-ai/mcpd/v2/internal/flags"
)

// RemoveCmd should be used to represent the 'remove' command.
type RemoveCmd struct {
	*cmd.BaseCmd
	cfgLoader config.Loader
}

// NewRemoveCmd creates a newly configured (Cobra) command.
func NewRemoveCmd(baseCmd *cmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	opts, err := cmdopts.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &RemoveCmd{
		BaseCmd:   baseCmd,
		cfgLoader: opts.ConfigLoader,
	}

	cobraCommand := &cobra.Command{
		Use:   "remove <server-name>",
		Short: "Removes an MCP server dependency from the project",
		Long:  "Removes an MCP server dependency from the project",
		RunE:  c.run,
	}

	return cobraCommand, nil
}

// run is configured (via NewRemoveCmd) to be called by the Cobra framework when the command is executed.
// It may return an error (or nil, when there is no error).
func (c *RemoveCmd) run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("server name is required and cannot be empty")
	}

	logger, err := c.Logger()
	if err != nil {
		return err
	}

	name := strings.TrimSpace(args[0])
	if name == "" {
		return fmt.Errorf("server name cannot be empty")
	}

	cfg, err := c.cfgLoader.Load(flags.ConfigFile)
	if err != nil {
		return err
	}

	err = cfg.RemoveServer(name)
	if err != nil {
		return err
	}

	logger.Debug("Server removed", "name", name)
	if _, err := fmt.Fprintf(
		cmd.OutOrStdout(),
		"✓ Removed server '%s'\n", name,
	); err != nil {
		return err
	}

	return nil
}
