package config

import (
	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"
)

func NewConfigCmd(logger hclog.Logger) *cobra.Command {
	cobraCmd := &cobra.Command{
		Use:   "config",
		Short: "Manages MCP server configuration.",
		Long:  "Manages MCP server configuration values and environment variable export.",
	}

	l := logger.Named("config")

	cobraCmd.AddCommand(NewSetArgsCmd(l))
	cobraCmd.AddCommand(NewSetEnvCmd(l))

	return cobraCmd
}
