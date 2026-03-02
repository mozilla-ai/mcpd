package volumes

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/internal/context"
	"github.com/mozilla-ai/mcpd/internal/flags"
)

// removeCmd handles removing volume mappings from MCP servers.
type removeCmd struct {
	*cmd.BaseCmd
	ctxLoader context.Loader
}

// NewRemoveCmd creates a new remove command for volume configuration.
func NewRemoveCmd(baseCmd *cmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	opts, err := cmdopts.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &removeCmd{
		BaseCmd:   baseCmd,
		ctxLoader: opts.ContextLoader,
	}

	cobraCmd := &cobra.Command{
		Use:   "remove <server-name> -- --<volume-name> [--<volume-name>...]",
		Short: "Remove volume mappings from an MCP server",
		Long: "Remove volume mappings from an MCP server in the runtime context " +
			"configuration file (e.g. " + flags.RuntimeFile + ").\n\n" +
			"Use -- to separate the server name from the volume names to remove.\n\n" +
			"Examples:\n" +
			"  # Remove a single volume mapping\n" +
			"  mcpd config volumes remove filesystem -- --workspace\n\n" +
			"  # Remove multiple volume mappings\n" +
			"  mcpd config volumes remove filesystem -- --workspace --gdrive",
		RunE: c.run,
		Args: validateRemoveArgs,
	}

	return cobraCmd, nil
}

// validateRemoveArgs validates the arguments for the remove command.
// It wraps validateRemoveArgsCore to extract the dash position from the cobra command.
func validateRemoveArgs(cmd *cobra.Command, args []string) error {
	return validateRemoveArgsCore(cmd.ArgsLenAtDash(), args)
}

// validateRemoveArgsCore validates the remove command arguments given the dash position and args slice.
func validateRemoveArgsCore(dashPos int, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("server-name is required")
	}
	if dashPos == -1 {
		return fmt.Errorf(
			"missing '--' separator: usage: mcpd config volumes remove <server-name> -- --<volume-name>",
		)
	}
	// -- at position 0 means no server name before it.
	if dashPos < 1 {
		return fmt.Errorf("server-name is required")
	}
	if strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("server-name is required")
	}
	if dashPos > 1 {
		return fmt.Errorf("too many arguments before --")
	}
	if len(args) < 2 {
		return fmt.Errorf("volume name(s) required after --")
	}
	return nil
}

// run executes the remove command, deleting volume mappings from the server config.
func (c *removeCmd) run(cobraCmd *cobra.Command, args []string) error {
	serverName := strings.TrimSpace(args[0])

	// volumeArgs contains everything after the -- separator.
	// validateRemoveArgs guarantees len(args) >= 2.
	volumeArgs := args[1:]
	volumeNames, err := parseRemoveArgs(volumeArgs)
	if err != nil {
		return err
	}

	cfg, err := c.ctxLoader.Load(flags.RuntimeFile)
	if err != nil {
		return fmt.Errorf("failed to load execution context config: %w", err)
	}

	// Get returns a value copy; safe to modify.
	server, ok := cfg.Get(serverName)
	if !ok {
		return fmt.Errorf("server '%s' not found in configuration", serverName)
	}

	working := maps.Clone(server.RawVolumes)
	if working == nil {
		working = context.VolumeExecutionContext{}
	}
	for _, name := range volumeNames {
		delete(working, name)
	}
	server = withVolumes(server, working)

	res, err := cfg.Upsert(server)
	if err != nil {
		return fmt.Errorf("error removing volumes for server '%s': %w", serverName, err)
	}

	sorted := slices.Clone(volumeNames)
	slices.Sort(sorted)

	out := cobraCmd.OutOrStdout()

	var msg string
	switch res {
	case context.Noop:
		msg = fmt.Sprintf("No changes — specified volumes not present on server '%s': %v", serverName, sorted)
	default:
		msg = fmt.Sprintf("✓ Volumes removed for server '%s' (operation: %s): %v", serverName, string(res), sorted)
	}

	if _, err := fmt.Fprintln(out, msg); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	return nil
}

// parseRemoveArgs parses volume name arguments in the format --name.
func parseRemoveArgs(args []string) ([]string, error) {
	names := make([]string, 0, len(args))

	for _, arg := range args {
		if !strings.HasPrefix(arg, "--") {
			return nil, fmt.Errorf("invalid volume name '%s': must start with --", arg)
		}

		name := strings.TrimSpace(strings.TrimPrefix(arg, "--"))
		if name == "" {
			return nil, fmt.Errorf("volume name cannot be empty in '%s'", arg)
		}

		names = append(names, name)
	}

	return names, nil
}
