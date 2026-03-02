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

func TestNewRemoveCmd(t *testing.T) {
	t.Parallel()

	base := &cmd.BaseCmd{}
	c, err := NewRemoveCmd(base)
	require.NoError(t, err)
	require.NotNil(t, c)

	require.True(t, strings.HasPrefix(c.Use, "remove "))
	require.Contains(t, c.Short, "Remove volume mappings")
	require.NotNil(t, c.RunE)
}

func TestRemoveCmd_run(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		serverName      string
		volumeArgs      []string
		existingServers map[string]context.ServerExecutionContext
		expectedOutput  string
		expectedError   string
		expectedVolumes context.VolumeExecutionContext
	}{
		{
			name:       "remove single volume",
			serverName: "filesystem",
			volumeArgs: []string{"--workspace"},
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
			expectedOutput:  "✓ Volumes removed for server 'filesystem' (operation: updated): [workspace]",
			expectedVolumes: context.VolumeExecutionContext{"gdrive": "/mcp/gdrive"},
		},
		{
			name:       "remove multiple volumes",
			serverName: "filesystem",
			volumeArgs: []string{"--workspace", "--gdrive"},
			existingServers: map[string]context.ServerExecutionContext{
				"filesystem": {
					Name: "filesystem",
					Volumes: context.VolumeExecutionContext{
						"workspace": "/Users/foo/repos",
						"gdrive":    "/mcp/gdrive",
						"data":      "vol",
					},
					RawVolumes: context.VolumeExecutionContext{
						"workspace": "/Users/foo/repos",
						"gdrive":    "/mcp/gdrive",
						"data":      "vol",
					},
				},
			},
			expectedOutput:  "✓ Volumes removed for server 'filesystem' (operation: updated): [gdrive workspace]",
			expectedVolumes: context.VolumeExecutionContext{"data": "vol"},
		},
		{
			name:       "remove all volumes",
			serverName: "filesystem",
			volumeArgs: []string{"--workspace"},
			existingServers: map[string]context.ServerExecutionContext{
				"filesystem": {
					Name:       "filesystem",
					Volumes:    context.VolumeExecutionContext{"workspace": "/Users/foo/repos"},
					RawVolumes: context.VolumeExecutionContext{"workspace": "/Users/foo/repos"},
				},
			},
			expectedOutput:  "✓ Volumes removed for server 'filesystem' (operation: deleted): [workspace]",
			expectedVolumes: context.VolumeExecutionContext{},
		},
		{
			name:       "remove nonexistent volume is a noop",
			serverName: "filesystem",
			volumeArgs: []string{"--nonexistent"},
			existingServers: map[string]context.ServerExecutionContext{
				"filesystem": {
					Name:       "filesystem",
					Volumes:    context.VolumeExecutionContext{"workspace": "/Users/foo/repos"},
					RawVolumes: context.VolumeExecutionContext{"workspace": "/Users/foo/repos"},
				},
			},
			expectedOutput:  "No changes — specified volumes not present on server 'filesystem': [nonexistent]",
			expectedVolumes: context.VolumeExecutionContext{"workspace": "/Users/foo/repos"},
		},
		{
			name:            "server not found",
			serverName:      "nonexistent",
			volumeArgs:      []string{"--workspace"},
			existingServers: map[string]context.ServerExecutionContext{},
			expectedError:   "server 'nonexistent' not found in configuration",
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
			removeCmd, err := NewRemoveCmd(base, cmdopts.WithContextLoader(loader))
			require.NoError(t, err)

			// Capture output.
			var output bytes.Buffer
			removeCmd.SetOut(&output)
			removeCmd.SetErr(&output)

			// Build args with -- separator.
			allArgs := append([]string{tc.serverName, "--"}, tc.volumeArgs...)
			removeCmd.SetArgs(allArgs)

			err = removeCmd.Execute()

			if tc.expectedError != "" {
				require.EqualError(t, err, tc.expectedError)
				return
			}

			require.NoError(t, err)

			actualOutput := strings.TrimSpace(output.String())
			require.Equal(t, tc.expectedOutput, actualOutput)

			// Verify the volumes were updated correctly.
			if tc.expectedVolumes != nil {
				require.Equal(t, tc.expectedVolumes, modifier.lastUpsert.Volumes)
				require.Equal(t, tc.expectedVolumes, modifier.lastUpsert.RawVolumes)
			}
		})
	}
}

func TestRemoveCmd_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		expectedError string
	}{
		{
			name:          "missing server name",
			args:          []string{"--", "--workspace"},
			expectedError: "server-name is required",
		},
		{
			name:          "empty server name",
			args:          []string{"", "--", "--workspace"},
			expectedError: "server-name is required",
		},
		{
			name:          "no volume names after separator",
			args:          []string{"server", "--"},
			expectedError: "volume name(s) required after --",
		},
		{
			name:          "missing -- separator",
			args:          []string{"server"},
			expectedError: "missing '--' separator: usage: mcpd config volumes remove <server-name> -- --<volume-name>",
		},
		{
			name:          "invalid volume name - no prefix",
			args:          []string{"server", "--", "workspace"},
			expectedError: "invalid volume name 'workspace': must start with --",
		},
		{
			name:          "empty volume name",
			args:          []string{"server", "--", "--"},
			expectedError: "volume name cannot be empty in '--'",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			modifier := &mockModifier{
				servers: map[string]context.ServerExecutionContext{},
			}
			loader := &mockLoader{modifier: modifier}

			base := &cmd.BaseCmd{}
			removeCmd, err := NewRemoveCmd(base, cmdopts.WithContextLoader(loader))
			require.NoError(t, err)

			removeCmd.SetArgs(tc.args)
			err = removeCmd.Execute()
			require.EqualError(t, err, tc.expectedError)
		})
	}
}

func TestRemoveCmd_LoaderError(t *testing.T) {
	t.Parallel()

	loader := &mockLoader{
		loadError: fmt.Errorf("failed to load"),
	}

	base := &cmd.BaseCmd{}
	removeCmd, err := NewRemoveCmd(base, cmdopts.WithContextLoader(loader))
	require.NoError(t, err)

	removeCmd.SetArgs([]string{"server", "--", "--workspace"})
	err = removeCmd.Execute()
	require.EqualError(t, err, "failed to load execution context config: failed to load")
}

func TestRemoveCmd_UpsertError(t *testing.T) {
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
	removeCmd, err := NewRemoveCmd(base, cmdopts.WithContextLoader(loader))
	require.NoError(t, err)

	removeCmd.SetArgs([]string{"server", "--", "--workspace"})
	err = removeCmd.Execute()
	require.EqualError(t, err, "error removing volumes for server 'server': upsert failed")
}

func TestValidateRemoveArgsCore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		dashPos       int
		expectedError string
	}{
		{
			name:    "valid args",
			args:    []string{"server", "--workspace"},
			dashPos: 1,
		},
		{
			name:          "no args at all",
			args:          []string{},
			dashPos:       -1,
			expectedError: "server-name is required",
		},
		{
			name:          "missing dash separator",
			args:          []string{"server"},
			dashPos:       -1,
			expectedError: "missing '--' separator: usage: mcpd config volumes remove <server-name> -- --<volume-name>",
		},
		{
			name:          "empty server name",
			args:          []string{"", "--workspace"},
			dashPos:       1,
			expectedError: "server-name is required",
		},
		{
			name:          "whitespace only server name",
			args:          []string{"   ", "--workspace"},
			dashPos:       1,
			expectedError: "server-name is required",
		},
		{
			name:          "dash at position 0",
			args:          []string{"--workspace"},
			dashPos:       0,
			expectedError: "server-name is required",
		},
		{
			name:          "too many args before dash",
			args:          []string{"server", "extra", "--workspace"},
			dashPos:       2,
			expectedError: "too many arguments before --",
		},
		{
			name:          "no volume names after dash",
			args:          []string{"server"},
			dashPos:       1,
			expectedError: "volume name(s) required after --",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := validateRemoveArgsCore(tc.dashPos, tc.args)

			if tc.expectedError != "" {
				require.EqualError(t, err, tc.expectedError)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestParseRemoveArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		expected      []string
		expectedError string
	}{
		{
			name:     "single volume name",
			args:     []string{"--workspace"},
			expected: []string{"workspace"},
		},
		{
			name:     "multiple volume names",
			args:     []string{"--workspace", "--gdrive"},
			expected: []string{"workspace", "gdrive"},
		},
		{
			name:          "missing -- prefix",
			args:          []string{"workspace"},
			expectedError: "invalid volume name 'workspace': must start with --",
		},
		{
			name:          "empty name after prefix",
			args:          []string{"--"},
			expectedError: "volume name cannot be empty in '--'",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := parseRemoveArgs(tc.args)

			if tc.expectedError != "" {
				require.EqualError(t, err, tc.expectedError)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}
