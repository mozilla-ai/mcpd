package daemon

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
)

type ListCmd struct {
	*cmd.BaseCmd
	cfgLoader config.Loader
	available bool
}

func NewListCmd(baseCmd *cmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	opts, err := cmdopts.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &ListCmd{
		BaseCmd:   baseCmd,
		cfgLoader: opts.ConfigLoader,
	}

	cobraCmd := &cobra.Command{
		Use:   "list",
		Short: "List daemon configuration",
		Long: `List daemon configuration from .mcpd.toml file.

Examples:
  mcpd config daemon list                # Show current configuration
  mcpd config daemon list --available   # Show all available configuration keys`,
		RunE: c.run,
		Args: cobra.NoArgs,
	}

	cobraCmd.Flags().
		BoolVar(&c.available, "available", false, "Show all available configuration keys with descriptions")

	return cobraCmd, nil
}

func (c *ListCmd) run(cmd *cobra.Command, args []string) error {
	if c.available {
		return c.showAvailableKeys(cmd)
	}

	cfg, err := c.LoadConfig(c.cfgLoader)
	if err != nil {
		return err
	}

	if cfg.Daemon == nil {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No daemon configuration found")
		return nil
	}

	// Use ConfigGetter to get all configuration
	allConfig, err := cfg.Daemon.Get()
	if err != nil {
		return err
	}

	return c.showConfig(cmd, allConfig, "daemon")
}

func (c *ListCmd) showConfig(cmd *cobra.Command, config any, prefix string) error {
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "[%s]\n", prefix)

	if configMap, ok := config.(map[string]any); ok {
		c.printConfigMap(cmd, configMap, prefix, "  ")
	}

	return nil
}

func (c *ListCmd) printConfigMap(cmd *cobra.Command, configMap map[string]any, prefix string, indent string) {
	for key, value := range configMap {
		switch v := value.(type) {
		case map[string]any:
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s[%s.%s]\n", indent, prefix, key)
			c.printConfigMap(cmd, v, fmt.Sprintf("%s.%s", prefix, key), indent+"  ")
		case []string:
			if len(v) > 0 {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s%s = %q\n", indent, key, v)
			}
		case bool:
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s%s = %t\n", indent, key, v)
		case string:
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s%s = %q\n", indent, key, v)
		default:
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s%s = %v\n", indent, key, v)
		}
	}
}

func (c *ListCmd) showAvailableKeys(cmd *cobra.Command) error {
	// Create a dummy daemon config to get the available keys
	daemonConfig := &config.DaemonConfig{}
	keys := daemonConfig.AvailableKeys()

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Available daemon configuration keys:")
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "")

	// Group keys by top-level section
	apiKeys := []config.ConfigKey{}
	mcpKeys := []config.ConfigKey{}

	for _, key := range keys {
		if strings.HasPrefix(key.Path, "api.") {
			apiKeys = append(apiKeys, key)
		} else if strings.HasPrefix(key.Path, "mcp.") {
			mcpKeys = append(mcpKeys, key)
		}
	}

	// Show API keys
	if len(apiKeys) > 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "API Configuration:")
		for _, key := range apiKeys {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %-30s %-12s %s\n", key.Path, "("+key.Type+")", key.Description)
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "")
	}

	// Show MCP keys
	if len(mcpKeys) > 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "MCP Configuration:")
		for _, key := range mcpKeys {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %-30s %-12s %s\n", key.Path, "("+key.Type+")", key.Description)
		}
	}

	return nil
}
