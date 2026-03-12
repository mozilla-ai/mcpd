package volumes

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/internal/context"
)

func TestNewClearCmd(t *testing.T) {
	t.Parallel()

	base := &cmd.BaseCmd{}
	c, err := NewClearCmd(base)
	require.NoError(t, err)
	require.NotNil(t, c)

	require.True(t, strings.HasPrefix(c.Use, "clear "))
	require.Contains(t, c.Short, "Clear")
	require.NotNil(t, c.RunE)

	forceFlag := c.Flags().Lookup("force")
	require.NotNil(t, forceFlag)
	require.Equal(t, "false", forceFlag.DefValue)
}

func TestClearCmd_run(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		serverName      string
		force           bool
		existingServers map[string]context.ServerExecutionContext
		expectedOutput  string
		expectedError   string
		expectedVolumes context.VolumeExecutionContext
	}{
		{
			name:       "clear all volumes with force",
			serverName: "filesystem",
			force:      true,
			existingServers: map[string]context.ServerExecutionContext{
				"filesystem": {
					Name: "filesystem",
					Volumes: context.VolumeExecutionContext{
						"workspace": "/Users/foo/repos",
						"gdrive":    "/mcp/gdrive",
					},
					RawVolumes: context.VolumeExecutionContext{
						"workspace": "/Users/foo/repos",
						"gdrive":    "/mcp/gdrive",
					},
				},
			},
			expectedOutput:  "✓ Volumes cleared for server 'filesystem' (operation: deleted)",
			expectedVolumes: context.VolumeExecutionContext{},
		},
		{
			name:       "clear without force returns error",
			serverName: "filesystem",
			force:      false,
			existingServers: map[string]context.ServerExecutionContext{
				"filesystem": {
					Name:       "filesystem",
					Volumes:    context.VolumeExecutionContext{"workspace": "/path"},
					RawVolumes: context.VolumeExecutionContext{"workspace": "/path"},
				},
			},
			expectedError: "this is a destructive operation. To clear all volumes for 'filesystem', " +
				"please re-run the command with the --force flag",
		},
		{
			name:            "server not found",
			serverName:      "nonexistent",
			force:           true,
			existingServers: map[string]context.ServerExecutionContext{},
			expectedError:   "server 'nonexistent' not found in configuration",
		},
		{
			name:       "clear server with no volumes is noop",
			serverName: "filesystem",
			force:      true,
			existingServers: map[string]context.ServerExecutionContext{
				"filesystem": {
					Name: "filesystem",
				},
			},
			expectedOutput:  "✓ Volumes cleared for server 'filesystem' (operation: noop)",
			expectedVolumes: context.VolumeExecutionContext{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			modifier := &mockModifier{
				servers: tc.existingServers,
			}
			loader := &mockLoader{modifier: modifier}

			base := &cmd.BaseCmd{}
			clearCmd, err := NewClearCmd(base, cmdopts.WithContextLoader(loader))
			require.NoError(t, err)

			var output bytes.Buffer
			clearCmd.SetOut(&output)
			clearCmd.SetErr(&output)

			args := []string{tc.serverName}
			if tc.force {
				args = append(args, "--force")
			}
			clearCmd.SetArgs(args)

			err = clearCmd.Execute()

			if tc.expectedError != "" {
				require.EqualError(t, err, tc.expectedError)
				return
			}

			require.NoError(t, err)

			actualOutput := strings.TrimSpace(output.String())
			require.Equal(t, tc.expectedOutput, actualOutput)

			if tc.expectedVolumes != nil {
				require.Equal(t, tc.expectedVolumes, modifier.lastUpsert.Volumes)
				require.Equal(t, tc.expectedVolumes, modifier.lastUpsert.RawVolumes)
			}
		})
	}
}

func TestClearCmd_LoaderError(t *testing.T) {
	t.Parallel()

	loader := &mockLoader{
		loadError: fmt.Errorf("failed to load"),
	}

	base := &cmd.BaseCmd{}
	clearCmd, err := NewClearCmd(base, cmdopts.WithContextLoader(loader))
	require.NoError(t, err)

	clearCmd.SetArgs([]string{"server", "--force"})
	err = clearCmd.Execute()
	require.EqualError(t, err, "failed to load execution context config: failed to load")
}

func TestClearCmd_UpsertError(t *testing.T) {
	t.Parallel()

	modifier := &mockModifier{
		servers: map[string]context.ServerExecutionContext{
			"server": {
				Name:       "server",
				Volumes:    context.VolumeExecutionContext{"workspace": "/path"},
				RawVolumes: context.VolumeExecutionContext{"workspace": "/path"},
			},
		},
		upsertError: fmt.Errorf("upsert failed"),
	}
	loader := &mockLoader{modifier: modifier}

	base := &cmd.BaseCmd{}
	clearCmd, err := NewClearCmd(base, cmdopts.WithContextLoader(loader))
	require.NoError(t, err)

	clearCmd.SetArgs([]string{"server", "--force"})
	err = clearCmd.Execute()
	require.EqualError(t, err, "error clearing volumes for server 'server': upsert failed")
}
