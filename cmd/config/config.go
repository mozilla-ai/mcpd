package config

import (
	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd-cli/v2/internal/cmd"
)

func NewConfigCmd(baseCmd *cmd.BaseCmd) *cobra.Command {
	cobraCmd := &cobra.Command{
		Use:   "config",
		Short: "Manages MCP server configuration.",
		Long:  "Manages MCP server configuration values and environment variable export.",
	}

	cobraCmd.AddCommand(NewSetArgsCmd(baseCmd))
	cobraCmd.AddCommand(NewSetEnvCmd(baseCmd))

	return cobraCmd
}
