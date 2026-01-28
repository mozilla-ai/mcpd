package daemon

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/internal/config"
	"github.com/mozilla-ai/mcpd/internal/context"
)

type SetCmd struct {
	*cmd.BaseCmd
	cfgLoader config.Loader
}

func NewSetCmd(baseCmd *cmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	opts, err := cmdopts.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &SetCmd{
		BaseCmd:   baseCmd,
		cfgLoader: opts.ConfigLoader,
	}

	cobraCmd := &cobra.Command{
		Use:   "set <key=value> [key=value ...]",
		Short: "Set daemon configuration values",
		Long: `Set daemon configuration values in .mcpd.toml file using dotted key notation.

Examples:
  mcpd config daemon set api.addr="0.0.0.0:9090"
  mcpd config daemon set api.timeout.shutdown="30s" mcp.interval.health="10s"
  mcpd config daemon set api.cors.enable=true api.cors.allow_origins="localhost:3000,app.example.com"`,
		RunE: c.run,
		Args: cobra.MinimumNArgs(1),
	}

	return cobraCmd, nil
}

func (c *SetCmd) run(cmd *cobra.Command, args []string) error {
	cfg, err := c.LoadConfig(c.cfgLoader)
	if err != nil {
		return err
	}

	// Initialize daemon config if needed
	if cfg.Daemon == nil {
		cfg.Daemon = &config.DaemonConfig{}
	}

	// Parse and set each key=value pair, collecting results
	type setResult struct {
		key    string
		result context.UpsertResult
	}
	var results []setResult

	for _, arg := range args {
		key, value, err := c.parseKeyValue(arg)
		if err != nil {
			return err
		}

		// CLI-level validation: empty values not allowed
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("empty value for key '%s', use 'mcpd config daemon remove %s' instead", key, key)
		}

		result, err := cfg.Daemon.Set(key, value)
		if err != nil {
			return err // Already has full path context from domain object
		}

		results = append(results, setResult{key: key, result: result})
	}

	// Save config first, then output results only if successful
	if err := cfg.SaveConfig(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Output results after successful save
	for _, res := range results {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "✓ Daemon config set for key '%s' (operation: %s)\n",
			res.key, string(res.result))
	}

	return nil
}

func (c *SetCmd) parseKeyValue(keyValue string) (string, string, error) {
	parts := strings.SplitN(keyValue, "=", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid format, expected key=value: %s", keyValue)
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	// Remove quotes if present
	if len(value) >= 2 &&
		((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
		value = value[1 : len(value)-1]
	}

	return key, value, nil
}
