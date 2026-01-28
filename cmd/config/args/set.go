package args

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/internal/config"
	"github.com/mozilla-ai/mcpd/internal/context"
	"github.com/mozilla-ai/mcpd/internal/flags"
)

type SetCmd struct {
	*cmd.BaseCmd
	ctxLoader  context.Loader
	mergeFlags bool
}

func NewSetCmd(baseCmd *cmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	opts, err := cmdopts.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &SetCmd{
		BaseCmd:   baseCmd,
		ctxLoader: opts.ContextLoader,
	}

	cobraCmd := &cobra.Command{
		Use:   "set <server-name> -- [positional-args...] [--flag=value...] [--bool-flag...]",
		Short: "Set or replace startup command line arguments for an MCP server",
		Long: `Set startup command line arguments for an MCP server in the runtime context 
configuration file (e.g. ` + "`~/.config/mcpd/secrets.dev.toml`" + `).

By default, this command completely replaces all existing arguments with the new ones provided.

Use the --merge-flags option to preserve existing flags while updating: it replaces all positional 
arguments with the new ones and merges flags (new flags override existing ones, 
non-conflicting flags are preserved).`,
		RunE: c.run,
		Args: func(cmd *cobra.Command, args []string) error {
			if cmd.ArgsLenAtDash() < 1 || strings.TrimSpace(args[0]) == "" {
				return fmt.Errorf("server-name is required")
			} else if cmd.ArgsLenAtDash() > 1 {
				return fmt.Errorf("too many arguments")
			} else if len(args) < 2 {
				return fmt.Errorf("argument(s) are required")
			}
			return nil
		},
	}

	cobraCmd.Flags().BoolVar(&c.mergeFlags, "merge-flags", false,
		"Replace positional args but merge flags (new flags override, others preserved)")

	return cobraCmd, nil
}

func (c *SetCmd) run(cmd *cobra.Command, args []string) error {
	serverName := strings.TrimSpace(args[0])
	if serverName == "" {
		return fmt.Errorf("server-name is required")
	}

	normalizedArgs := config.ProcessAllArgs(args[1:])
	cfg, err := c.ctxLoader.Load(flags.RuntimeFile)
	if err != nil {
		return fmt.Errorf("failed to load execution context config: %w", err)
	}

	server, exists := cfg.Get(serverName)
	if !exists {
		server.Name = serverName
	}

	newArgs := normalizedArgs
	if c.mergeFlags {
		newArgs = config.MergeArgsWithPositionalHandling(server.Args, normalizedArgs)
	}

	server.Args = newArgs
	if len(server.Env) == 0 {
		server.Env = map[string]string{}
	}

	res, err := cfg.Upsert(server)
	if err != nil {
		return fmt.Errorf("error setting arguments for server '%s': %w", serverName, err)
	}

	operation := "replaced"
	if c.mergeFlags {
		operation = "merged (flags only)"
	}

	if _, err := fmt.Fprintf(
		cmd.OutOrStdout(),
		"✓ Startup arguments %s for server '%s' (operation: %s): %v\n", operation, serverName, string(res), normalizedArgs,
	); err != nil {
		return err
	}

	return nil
}
