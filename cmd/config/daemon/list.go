package daemon

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/internal/config"
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

	// Use Getter to get all configuration
	allConfig, err := cfg.Daemon.Get()
	if err != nil {
		return err
	}

	return c.showConfig(cmd, allConfig, "daemon")
}

func (c *ListCmd) showConfig(cmd *cobra.Command, config any, prefix string) error {
	// Flatten the config into dotted key-value pairs
	flatConfig := make(map[string]any)
	c.flattenConfig(config, "", flatConfig)

	// Sort the keys
	var keys []string
	for key := range flatConfig {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Print the sorted key-value pairs
	for _, key := range keys {
		value := flatConfig[key]
		c.printKeyValue(cmd, key, value)
	}

	return nil
}

// flattenConfig recursively flattens a nested configuration map into dotted key-value pairs.
// The prefix parameter is used to build the full dotted path for nested keys.
func (c *ListCmd) flattenConfig(value any, prefix string, result map[string]any) {
	prefix = strings.ToLower(strings.TrimSpace(prefix))
	switch v := value.(type) {
	case map[string]any:
		for key, val := range v {
			newKey := key
			if prefix != "" {
				newKey = prefix + "." + key
			}
			c.flattenConfig(val, newKey, result)
		}
	default:
		if prefix != "" {
			result[prefix] = value
		}
	}
}

// printKeyValue formats and prints a single configuration key-value pair with appropriate type formatting.
func (c *ListCmd) printKeyValue(cmd *cobra.Command, key string, value any) {
	switch v := value.(type) {
	case []string:
		if len(v) > 0 {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s = %q\n", key, v)
		}
	case bool:
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s = %t\n", key, v)
	case string:
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s = %q\n", key, v)
	default:
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s = %v\n", key, v)
	}
}

// showAvailableKeys displays all available daemon configuration keys with their types and descriptions.
func (c *ListCmd) showAvailableKeys(cmd *cobra.Command) error {
	// Create a dummy daemon config to get the available keys
	daemonConfig := &config.DaemonConfig{}
	keys := daemonConfig.AvailableKeys()

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Available daemon configuration keys:")
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "")

	// Group keys by top-level section
	var apiKeys []config.SchemaKey
	var mcpKeys []config.SchemaKey

	for _, key := range keys {
		if strings.HasPrefix(key.Path, "api.") {
			apiKeys = append(apiKeys, key)
		} else if strings.HasPrefix(key.Path, "mcp.") {
			mcpKeys = append(mcpKeys, key)
		}
	}

	// Sort keys within each section
	sort.Slice(apiKeys, func(i, j int) bool {
		return apiKeys[i].Path < apiKeys[j].Path
	})
	sort.Slice(mcpKeys, func(i, j int) bool {
		return mcpKeys[i].Path < mcpKeys[j].Path
	})

	// Show API keys
	if len(apiKeys) > 0 {
		c.showKeySection(cmd, "API Configuration:", apiKeys)
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "")
	}

	// Show MCP keys
	if len(mcpKeys) > 0 {
		c.showKeySection(cmd, "MCP Configuration:", mcpKeys)
	}

	return nil
}

// showKeySection displays a section of configuration keys with consistent formatting.
func (c *ListCmd) showKeySection(cmd *cobra.Command, title string, keys []config.SchemaKey) {
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), title)
	for _, key := range keys {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %-35s %-12s %s\n", key.Path, "("+key.Type+")", key.Description)
	}
}
