package config

import (
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "config",
	Short: "Manages MCP server configuration.",
	Long:  "Manages MCP server configuration values and environment variable export.",
}

func init() {
	// TODO: Re-add subcommands.
	// Add subcommands to the config command.
	// Cmd.AddCommand(exportEnvCmd)
	// Cmd.AddCommand(setCmd)
}
