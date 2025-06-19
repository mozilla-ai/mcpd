package cmd

import (
	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
)

func NewConfigCmd(baseCmd *cmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	cobraCmd := &cobra.Command{
		Use:   "config",
		Short: "Manages MCP server configuration.",
		Long:  "Manages MCP server configuration values and environment variable export.",
	}

	fns := []createCmdFunc{
		NewSetArgsCmd,
		NewSetEnvCmd,
	}

	for _, fn := range fns {
		tempCmd, err := fn(baseCmd, opt...)
		if err != nil {
			return nil, err
		}
		cobraCmd.AddCommand(tempCmd)
	}

	return cobraCmd, nil
}
