package server

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd-cli/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd-cli/v2/internal/config"
)

// AddCmd should be used to represent the 'add' command.
type AddCmd struct {
	*cmd.BaseCmd
	Version string
	Tools   []string
}

// NewAddCmd creates a newly configured (Cobra) command.
func NewAddCmd(logger hclog.Logger) *cobra.Command {
	c := &AddCmd{
		BaseCmd: &cmd.BaseCmd{Logger: logger},
	}

	cobraCommand := &cobra.Command{
		Use:   "add <server_name>",
		Short: "Adds an MCP server dependency to the project.",
		Long:  c.longDescription(),
		RunE:  c.run,
	}

	cobraCommand.Flags().StringVar(
		&c.Version,
		"version",
		"latest",
		"Specify the version of the server package",
	)
	cobraCommand.Flags().StringArrayVar(
		&c.Tools,
		"tool",
		nil,
		"Optional, when specified limits the available tools on the server (can be repeated)",
	)

	return cobraCommand
}

// longDescription returns the long version of the command description.
func (c *AddCmd) longDescription() string {
	return `Adds an MCP server dependency to the project. 
mcpd will search the registry for the server and attempt to return information on the version specified, 
or 'latest' if no version specified.`
}

// run is configured (via NewAddCmd) to be called by the Cobra framework when the command is executed.
// It may return an error (or nil, when there is no error).
func (c *AddCmd) run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("server name is required and cannot be empty")
	}

	name := strings.TrimSpace(args[0])
	if name == "" {
		return fmt.Errorf("server name cannot be empty")
	}

	// TODO: Make an actual call to the mcpd registry to get information here.
	// Currently, we just fake the response here so we can deal with the config file.
	pkg := fmt.Sprintf("modelcontextprotocol/%s@%s", name, c.Version)

	entry := config.ServerEntry{
		Name:    name,
		Package: pkg,
		Tools:   c.Tools,
	}

	if err := config.AddServer(entry); err != nil {
		return err
	}

	// User-friendly output + logging
	fmt.Fprintf(cmd.OutOrStdout(), "✓ Added server '%s' (version: %s)\n", name, c.Version)
	if len(c.Tools) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "  Tools: %s\n", strings.Join(c.Tools, ", "))
	}
	c.Logger.Debug("Server added", "name", name, "version", c.Version, "tools", c.Tools)

	return nil
}
