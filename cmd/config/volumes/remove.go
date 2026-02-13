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

// removeCmd handles removing volume mappings for MCP servers.
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
		Use:   "remove <server-name> VOLUME [VOLUME ...]",
		Short: "Remove volume mappings for an MCP server",
		Long: "Remove volume mappings for a specified MCP server from the runtime context " +
			"configuration file (e.g. " + flags.RuntimeFile + ").\n\n" +
			"Volume names are positional arguments (not flags), unlike the 'set' command\n" +
			"which uses --name=value flag syntax and requires a -- separator.\n\n" +
			"Examples:\n" +
			"  # Remove a single volume mapping\n" +
			"  mcpd config volumes remove filesystem workspace\n\n" +
			"  # Remove multiple volume mappings\n" +
			"  mcpd config volumes remove filesystem workspace gdrive",
		RunE: c.run,
		// Unlike set, remove uses positional args so cobra.MinimumNArgs handles
		// arity; parseRemoveArgs handles semantic validation in run().
		Args: cobra.MinimumNArgs(2),
	}

	return cobraCmd, nil
}

// parseRemoveArgs validates and parses the remove command arguments.
// It returns the trimmed server name and a deduplicated list of volume names.
func parseRemoveArgs(args []string) (string, []string, error) {
	if len(args) < 2 {
		return "", nil, fmt.Errorf("at least a server name and one volume name are required")
	}

	serverName := strings.TrimSpace(args[0])
	if serverName == "" {
		return "", nil, fmt.Errorf("server-name is required")
	}

	// Deduplicate and validate volume names.
	// args[1:] is safe: guard above guarantees len(args) >= 2.
	seen := make(map[string]struct{}, len(args)-1)
	volumeNames := make([]string, 0, len(args)-1)
	for _, name := range args[1:] {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			return "", nil, fmt.Errorf("volume name argument cannot be empty")
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		volumeNames = append(volumeNames, trimmed)
	}

	return serverName, volumeNames, nil
}

// run executes the remove command, deleting specified volume mappings from the server config.
func (c *removeCmd) run(cmd *cobra.Command, args []string) error {
	serverName, volumeNames, err := parseRemoveArgs(args)
	if err != nil {
		return err
	}

	cfg, err := c.ctxLoader.Load(flags.RuntimeFile)
	if err != nil {
		return fmt.Errorf("failed to load execution context config: %w", err)
	}

	server, exists := cfg.Get(serverName)
	if !exists {
		return fmt.Errorf("server '%s' not found in configuration", serverName)
	}

	// Clone into a working map to avoid mutating the config's internal state.
	// Use RawVolumes as the source of truth to avoid persisting expanded
	// environment variables back to disk. Fall back to Volumes if RawVolumes is nil.
	working := maps.Clone(server.RawVolumes)
	if working == nil {
		working = maps.Clone(server.Volumes)
	}
	if working == nil {
		working = context.VolumeExecutionContext{}
	}

	removed := make([]string, 0, len(volumeNames))
	var notFound []string
	for _, name := range volumeNames {
		if _, ok := working[name]; ok {
			delete(working, name)
			removed = append(removed, name)
		} else {
			notFound = append(notFound, name)
		}
	}

	if len(removed) == 0 {
		return fmt.Errorf("no matching volumes found for server '%s': %v", serverName, volumeNames)
	}

	// Assign the working map back so Upsert persists unexpanded values.
	// RawVolumes is the single source of truth; Volumes is derived.
	server.RawVolumes = working
	server.Volumes = maps.Clone(working)

	res, err := cfg.Upsert(server)
	if err != nil {
		return fmt.Errorf("error removing volumes for server '%s': %w", serverName, err)
	}

	slices.Sort(removed)
	out := cmd.OutOrStdout()
	if _, err := fmt.Fprintf(
		out,
		"✓ Volumes removed for server '%s' (operation: %s): %v\n", serverName, string(res), removed,
	); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	if len(notFound) > 0 {
		slices.Sort(notFound)
		if _, err := fmt.Fprintf(out, "  Not found (skipped): %v\n", notFound); err != nil {
			return fmt.Errorf("failed to write output: %w", err)
		}
	}

	return nil
}
