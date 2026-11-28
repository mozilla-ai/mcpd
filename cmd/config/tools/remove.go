package tools

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/internal/config"
	"github.com/mozilla-ai/mcpd/internal/domain"
	"github.com/mozilla-ai/mcpd/internal/flags"
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

	// A server name is matched without regard to case, so the name works here as it does in a daemon URL.
	wanted := domain.ServerName(serverName).Key()
	for _, srv := range cfg.ListServers() {
		if domain.ServerName(srv.Name).Key() != wanted {
			continue
		}

		removed := map[string]struct{}{}

		for k := range toolsMap {
			if idx := slices.Index(srv.Tools, k); idx >= 0 {
				srv.Tools = slices.Delete(srv.Tools, idx, idx+1)
				removed[k] = struct{}{}
			}
		}

		// Validate that at least one tool remains.
		if len(srv.Tools) == 0 {
			return fmt.Errorf("cannot remove all tools from server '%s'\n"+
				"To remove the server instead use: mcpd remove %s", srv.Name, srv.Name)
		}

		// The stored name is used rather than the supplied one, because the lookup above matched without
		// regard to case and the configuration holds only the spelling it was written with.
		err = cfg.UpdateServer(srv)
		if err != nil {
			return fmt.Errorf("unable to remove tools for server '%s': %w", srv.Name, err)
		}

		msg := "✓ Command completed successfully, no tools required removal\n"
		if len(removed) > 0 {
			msg = fmt.Sprintf("✓ Tools removed for server '%s': %v\n", srv.Name, slices.Collect(maps.Keys(removed)))
		}

		_, err = fmt.Fprint(cmd.OutOrStdout(), msg)
		if err != nil {
			return err
		}

		break
	}

	return nil
}
