package daemon

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
)

type ValidateCmd struct {
	*cmd.BaseCmd
	cfgLoader config.Loader
}

func NewValidateCmd(baseCmd *cmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	opts, err := cmdopts.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &ValidateCmd{
		BaseCmd:   baseCmd,
		cfgLoader: opts.ConfigLoader,
	}

	cobraCmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate daemon configuration",
		Long:  `Validate daemon configuration in .mcpd.toml file`,
		RunE:  c.run,
		Args:  cobra.NoArgs,
	}

	return cobraCmd, nil
}

func (c *ValidateCmd) run(cmd *cobra.Command, _ []string) error {
	cfg, err := c.LoadConfig(c.cfgLoader)
	if err != nil {
		return err
	}

	// Validate daemon configuration using domain method
	if cfg.Daemon == nil {
		return fmt.Errorf("no daemon configuration found")
	}
	if err := cfg.Daemon.Validate(); err != nil {
		_, _ = fmt.Fprintf(cmd.OutOrStderr(), "✗ Daemon configuration validation failed: %v\n", err)
		return err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Daemon configuration is valid\n")
	return nil
}
