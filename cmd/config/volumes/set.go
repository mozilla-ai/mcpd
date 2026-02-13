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

// setCmd handles setting volume mappings for MCP servers.
type setCmd struct {
	*cmd.BaseCmd
	ctxLoader context.Loader
}

// NewSetCmd creates a new set command for volume configuration.
func NewSetCmd(baseCmd *cmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	opts, err := cmdopts.NewOptions(opt...)
	if err != nil {
		return nil, err
	}

	c := &setCmd{
		BaseCmd:   baseCmd,
		ctxLoader: opts.ContextLoader,
	}

	cobraCmd := &cobra.Command{
		Use:   "set <server-name> -- --<volume-name>=<host-path> [--<volume-name>=<host-path>...]",
		Short: "Set or update volume mappings for an MCP server",
		Long: "Set volume mappings for an MCP server in the runtime context " +
			"configuration file (e.g. " + flags.RuntimeFile + ").\n\n" +
			"Volume mappings associate volume names with host paths or named Docker volumes.\n" +
			"Use -- to separate the server name from the volume mappings.\n\n" +
			"Examples:\n" +
			"  # Set a single volume mapping\n" +
			"  mcpd config volumes set filesystem -- --workspace=/Users/foo/repos/mcpd\n\n" +
			"  # Set multiple volume mappings\n" +
			"  mcpd config volumes set filesystem -- --workspace=\"/Users/foo/repos\" --gdrive=\"/mcp/gdrive\"\n\n" +
			"  # Use a named Docker volume\n" +
			"  mcpd config volumes set myserver -- --data=my-named-volume",
		RunE: c.run,
		Args: validateSetArgs,
	}

	return cobraCmd, nil
}

// validateSetArgs validates the arguments for the set command.
// It wraps validateArgs to extract the dash position from the cobra command.
func validateSetArgs(cmd *cobra.Command, args []string) error {
	return validateArgs(cmd.ArgsLenAtDash(), args)
}

// validateArgs validates the set command arguments given the dash position and args slice.
func validateArgs(dashPos int, args []string) error {
	// No args at all.
	if len(args) == 0 {
		return fmt.Errorf("server-name is required")
	}
	// Args provided but no -- separator (user forgot --).
	if dashPos == -1 {
		return fmt.Errorf(
			"missing '--' separator: usage: mcpd config volumes set <server-name> -- --<volume>=<path>",
		)
	}
	// -- at position 0 (no server name before it) or server name is empty.
	if dashPos < 1 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("server-name is required")
	}
	if dashPos > 1 {
		return fmt.Errorf("too many arguments before --")
	}
	if len(args) < 2 {
		return fmt.Errorf("volume mapping(s) required after --")
	}
	return nil
}

// run executes the set command, parsing volume mappings and updating the server config.
func (c *setCmd) run(cmd *cobra.Command, args []string) error {
	serverName := strings.TrimSpace(args[0])

	// volumeArgs contains everything after the -- separator.
	// validateSetArgs guarantees len(args) >= 2.
	volumeArgs := args[1:]
	volumeMap, err := parseVolumeArgs(volumeArgs)
	if err != nil {
		return err
	}

	cfg, err := c.ctxLoader.Load(flags.RuntimeFile)
	if err != nil {
		return fmt.Errorf("failed to load execution context config: %w", err)
	}

	server, exists := cfg.Get(serverName)
	if !exists {
		server.Name = serverName
	}

	// Use RawVolumes as the source of truth to avoid persisting expanded
	// environment variables (e.g. ${MCPD__...} placeholders) back to disk.
	if server.RawVolumes == nil {
		server.RawVolumes = context.VolumeExecutionContext{}
	}

	maps.Copy(server.RawVolumes, volumeMap)

	// Sync Volumes from RawVolumes so Upsert persists unexpanded values.
	server.Volumes = server.RawVolumes

	res, err := cfg.Upsert(server)
	if err != nil {
		return fmt.Errorf("error setting volumes for server '%s': %w", serverName, err)
	}

	volumeNames := slices.Collect(maps.Keys(volumeMap))
	slices.Sort(volumeNames)
	if _, err := fmt.Fprintf(
		cmd.OutOrStdout(),
		"✓ Volumes set for server '%s' (operation: %s): %v\n", serverName, string(res), volumeNames,
	); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	return nil
}

// parseVolumeArgs parses volume arguments in the format --name=path or --name="path".
func parseVolumeArgs(args []string) (map[string]string, error) {
	volumes := make(map[string]string, len(args))

	for _, arg := range args {
		originalArg := arg

		// Expect format: --name=path
		if !strings.HasPrefix(arg, "--") {
			return nil, fmt.Errorf("invalid volume format '%s': must start with --", arg)
		}

		// Remove the -- prefix for parsing.
		arg = strings.TrimPrefix(arg, "--")

		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid volume format '%s': expected --<volume-name>=<host-path>", originalArg)
		}

		name := strings.TrimSpace(parts[0])
		path := strings.TrimSpace(parts[1])

		if name == "" {
			return nil, fmt.Errorf("volume name cannot be empty in '%s'", originalArg)
		}

		if path == "" {
			return nil, fmt.Errorf("volume path cannot be empty for volume '%s'", name)
		}

		// Remove surrounding quotes if present.
		path = trimQuotes(path)
		if path == "" {
			return nil, fmt.Errorf("volume path cannot be empty for volume '%s'", name)
		}

		volumes[name] = path
	}

	return volumes, nil
}

// trimQuotes removes surrounding single or double quotes from a string.
func trimQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
