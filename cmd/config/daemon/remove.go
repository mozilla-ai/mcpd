package daemon

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/internal/config"
)

type RemoveCmd struct {
	*cmd.BaseCmd
	cfgLoader config.Loader
}

func NewRemoveCmd(baseCmd *cmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	opts, err := cmdopts.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &RemoveCmd{
		BaseCmd:   baseCmd,
		cfgLoader: opts.ConfigLoader,
	}

	cobraCmd := &cobra.Command{
		Use:   "remove <key> [key ...]",
		Short: "Remove daemon configuration values",
		Long: `Remove specific daemon configuration values from .mcpd.toml file using dotted key notation.

Examples:
  mcpd config daemon remove api.addr
  mcpd config daemon remove api.cors.enable api.cors.allow_origins
  mcpd config daemon remove mcp.timeout.health`,
		RunE: c.run,
		Args: cobra.MinimumNArgs(1),
	}

	return cobraCmd, nil
}

func (c *RemoveCmd) run(cmd *cobra.Command, args []string) error {
	cfg, err := c.LoadConfig(c.cfgLoader)
	if err != nil {
		return err
	}

	if cfg.Daemon == nil {
		return fmt.Errorf("no daemon configuration found")
	}

	// Remove each specified key by setting to empty value
	for _, key := range args {
		_, err := cfg.Daemon.Set(key, "") // Internal use with empty value
		if err != nil {
			return err // Already has full path context from domain object
		}

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Removed daemon config '%s'\n", key)
	}

	// Save config
	if err := cfg.SaveConfig(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}
