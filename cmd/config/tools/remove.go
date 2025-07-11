package tools

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/v2/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
	"github.com/mozilla-ai/mcpd/v2/internal/flags"
)

type RemoveCmd struct {
	*cmd.BaseCmd
	cfgLoader config.Loader
	Tools     []string
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

	// mcpd config tools remove time TOOL [TOOL ...]
	cobraCmd := &cobra.Command{
		Use:   "remove <server-name> TOOL [TOOL ...]",
		Short: "Remove allowed-listed tools for an MCP server from configuration",
		Long: "Remove allowed-listed tools for an MCP server from configuration, " +
			"if the specified tools are present in config they will be removed",
		RunE: c.run,
		Args: cobra.MinimumNArgs(2), // server-name + TOOL ...
	}

	return cobraCmd, nil
}

func (c *RemoveCmd) run(cmd *cobra.Command, args []string) error {
	serverName := strings.TrimSpace(args[0])
	if serverName == "" {
		return fmt.Errorf("server-name is required")
	}

	tools := args[1:]
	toolsMap := make(map[string]struct{}, len(tools))
	for _, key := range tools {
		key = strings.TrimSpace(key)
		toolsMap[key] = struct{}{}
	}

	cfg, err := c.cfgLoader.Load(flags.ConfigFile)
	if err != nil {
		return err
	}

	for _, srv := range cfg.ListServers() {
		if srv.Name != serverName {
			continue
		}

		removed := map[string]struct{}{}

		for k := range toolsMap {
			if idx := slices.Index(srv.Tools, k); idx >= 0 {
				srv.Tools = slices.Delete(srv.Tools, idx, idx+1)
				removed[k] = struct{}{}
			}
		}

		// Update server in config by removing and re-adding.
		err = cfg.RemoveServer(serverName)
		if err != nil {
			return fmt.Errorf("error removing server, unable to remove tools for server %s: %w", serverName, err)
		}

		err = cfg.AddServer(srv)
		if err != nil {
			return fmt.Errorf("error adding server, unable to remove tools for server %s: %w", serverName, err)
		}

		msg := "✓ Command completed successfully, no tools required removal\n"
		if len(removed) > 0 {
			msg = fmt.Sprintf("✓ Tools removed for server '%s': %v\n", serverName, slices.Collect(maps.Keys(removed)))
		}

		_, err = fmt.Fprint(cmd.OutOrStdout(), msg)
		if err != nil {
			return err
		}

		break
	}

	return nil
}
