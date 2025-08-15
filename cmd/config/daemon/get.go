package daemon

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
)

type GetCmd struct {
	*cmd.BaseCmd
	cfgLoader config.Loader
}

func NewGetCmd(baseCmd *cmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	opts, err := cmdopts.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &GetCmd{
		BaseCmd:   baseCmd,
		cfgLoader: opts.ConfigLoader,
	}

	cobraCmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get daemon configuration value",
		Long: `Get a specific daemon configuration value from .mcpd.toml file using dotted key notation.

Examples:
  mcpd config daemon get api.addr
  mcpd config daemon get api.cors.enable
  mcpd config daemon get mcp.timeout.health`,
		RunE: c.run,
		Args: cobra.ExactArgs(1),
	}

	return cobraCmd, nil
}

func (c *GetCmd) run(cmd *cobra.Command, args []string) error {
	cfg, err := c.LoadConfig(c.cfgLoader)
	if err != nil {
		return err
	}

	if cfg.Daemon == nil {
		return fmt.Errorf("no daemon configuration found")
	}

	// Split dotted notation into keys for variadic Get
	key := args[0]
	keys := strings.Split(key, ".")

	value, err := cfg.Daemon.Get(keys...)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), c.formatValue(value))
	return nil
}

func (c *GetCmd) formatValue(value any) string {
	switch v := value.(type) {
	case *config.Duration:
		if v == nil {
			return ""
		}
		return v.String()
	case config.Duration:
		return v.String()
	case []string:
		if len(v) == 0 {
			return "[]"
		}
		return strings.Join(v, ",")
	case *string:
		if v == nil {
			return ""
		}
		return *v
	case *bool:
		if v == nil {
			return ""
		}
		return fmt.Sprintf("%t", *v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
