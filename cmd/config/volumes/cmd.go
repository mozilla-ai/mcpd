package volumes

import (
	"fmt"
	"maps"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mozilla-ai/mcpd/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/internal/context"
)

// NewCmd creates a new volumes command with its sub-commands.
func NewCmd(baseCmd *cmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error) {
	cobraCmd := &cobra.Command{
		Use:   "volumes",
		Short: "Manages volume configuration for MCP servers",
		Long: "Manages Docker volume configuration for MCP servers, " +
			"dealing with setting, removing, clearing and listing volume mappings.",
	}

	// Sub-commands for: mcpd config volumes
	fns := []func(baseCmd *cmd.BaseCmd, opt ...cmdopts.CmdOption) (*cobra.Command, error){
		NewListCmd,   // list
		NewRemoveCmd, // remove
		NewSetCmd,    // set
	}

	for _, fn := range fns {
		tempCmd, err := fn(baseCmd, opt...)
		if err != nil {
			return nil, err
		}
		cobraCmd.AddCommand(tempCmd)
	}

	return cobraCmd, nil
}

// stripDashPrefix validates that arg starts with "--" and returns the content after the prefix.
func stripDashPrefix(arg string) (string, error) {
	if !strings.HasPrefix(arg, "--") {
		return "", fmt.Errorf("'%s': must start with --", arg)
	}
	return strings.TrimPrefix(arg, "--"), nil
}

// withVolumes returns a new ServerExecutionContext with both volume fields
// set to unexpanded values. Volumes (the TOML-serialized field) preserves
// env var references on disk. RawVolumes is kept in sync for Equals/IsEmpty
// comparisons during Upsert.
func withVolumes(
	server context.ServerExecutionContext,
	working context.VolumeExecutionContext,
) context.ServerExecutionContext {
	server.RawVolumes = maps.Clone(working)
	server.Volumes = maps.Clone(working)
	return server
}
