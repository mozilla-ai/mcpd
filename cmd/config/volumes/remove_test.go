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

	require.True(t, strings.HasPrefix(c.Use, "remove"))
	require.Contains(t, c.Short, "Remove volume mappings")
	require.NotNil(t, c.RunE)
}

func TestParseRemoveArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		args            []string
		expectedServer  string
		expectedVolumes []string
		expectedError   string
	}{
		{
			name:            "single volume",
			args:            []string{"server", "workspace"},
			expectedServer:  "server",
			expectedVolumes: []string{"workspace"},
		},
		{
			name:            "multiple volumes",
			args:            []string{"server", "workspace", "gdrive"},
			expectedServer:  "server",
			expectedVolumes: []string{"workspace", "gdrive"},
		},
		{
			name:            "trims whitespace from server name",
			args:            []string{"  server  ", "workspace"},
			expectedServer:  "server",
			expectedVolumes: []string{"workspace"},
		},
		{
			name:            "trims whitespace from volume names",
			args:            []string{"server", "  workspace  "},
			expectedServer:  "server",
			expectedVolumes: []string{"workspace"},
		},
		{
			name:            "deduplicates volume names",
			args:            []string{"server", "workspace", "workspace", "gdrive"},
			expectedServer:  "server",
			expectedVolumes: []string{"workspace", "gdrive"},
		},
		{
			name:          "empty server name",
			args:          []string{"", "workspace"},
			expectedError: "server-name is required",
		},
		{
			name:          "whitespace-only server name",
			args:          []string{"   ", "workspace"},
			expectedError: "server-name is required",
		},
		{
			name:          "empty volume name",
			args:          []string{"server", ""},
			expectedError: "volume name argument cannot be empty",
		},
		{
			name:          "whitespace-only volume name",
			args:          []string{"server", "   "},
			expectedError: "volume name argument cannot be empty",
		},
		{
			name:          "empty volume name at second position",
			args:          []string{"server", "workspace", ""},
			expectedError: "volume name argument cannot be empty",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			serverName, volumeNames, err := parseRemoveArgs(tc.args)

			if tc.expectedError != "" {
				require.EqualError(t, err, tc.expectedError)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expectedServer, serverName)
			require.Equal(t, tc.expectedVolumes, volumeNames)
		})
	}
}

func TestRemoveCmd_run(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		args            []string
		existingServers map[string]context.ServerExecutionContext
		expectedOutput  string
		expectedError   string
		expectedVolumes context.VolumeExecutionContext
	}{
		{
			name: "remove single volume",
			args: []string{"test-server", "workspace"},
			existingServers: map[string]context.ServerExecutionContext{
				"test-server": {
					Name:       "test-server",
					Volumes:    context.VolumeExecutionContext{"workspace": "/path", "gdrive": "/mcp/gdrive"},
					RawVolumes: context.VolumeExecutionContext{"workspace": "/path", "gdrive": "/mcp/gdrive"},
				},
			},
			expectedOutput:  "✓ Volumes removed for server 'test-server' (operation: updated): [workspace]",
			expectedVolumes: context.VolumeExecutionContext{"gdrive": "/mcp/gdrive"},
		},
		{
			name: "remove multiple volumes",
			args: []string{"test-server", "workspace", "gdrive"},
			existingServers: map[string]context.ServerExecutionContext{
				"test-server": {
					Name: "test-server",
					Volumes: context.VolumeExecutionContext{
						"workspace": "/path",
						"gdrive":    "/mcp/gdrive",
						"data":      "/data",
					},
					RawVolumes: context.VolumeExecutionContext{
						"workspace": "/path",
						"gdrive":    "/mcp/gdrive",
						"data":      "/data",
					},
				},
			},
			expectedOutput:  "✓ Volumes removed for server 'test-server' (operation: updated): [gdrive workspace]",
			expectedVolumes: context.VolumeExecutionContext{"data": "/data"},
		},
		{
			name: "remove all volumes",
			args: []string{"test-server", "workspace"},
			existingServers: map[string]context.ServerExecutionContext{
				"test-server": {
					Name:       "test-server",
					Volumes:    context.VolumeExecutionContext{"workspace": "/path"},
					RawVolumes: context.VolumeExecutionContext{"workspace": "/path"},
				},
			},
			expectedOutput:  "✓ Volumes removed for server 'test-server' (operation:",
			expectedVolumes: context.VolumeExecutionContext{},
		},
		{
			name: "remove with nil RawVolumes falls back to Volumes",
			args: []string{"test-server", "workspace"},
			existingServers: map[string]context.ServerExecutionContext{
				"test-server": {
					Name:    "test-server",
					Volumes: context.VolumeExecutionContext{"workspace": "/path", "gdrive": "/mcp/gdrive"},
				},
			},
			expectedOutput:  "✓ Volumes removed for server 'test-server' (operation: updated): [workspace]",
			expectedVolumes: context.VolumeExecutionContext{"gdrive": "/mcp/gdrive"},
		},
		{
			name: "server not found",
			args: []string{"nonexistent", "workspace"},
			existingServers: map[string]context.ServerExecutionContext{
				"test-server": {
					Name:    "test-server",
					Volumes: context.VolumeExecutionContext{"workspace": "/path"},
				},
			},
			expectedError: "server 'nonexistent' not found in configuration",
		},
		{
			name: "no matching volumes",
			args: []string{"test-server", "nonexistent"},
			existingServers: map[string]context.ServerExecutionContext{
				"test-server": {
					Name:       "test-server",
					Volumes:    context.VolumeExecutionContext{"workspace": "/path"},
					RawVolumes: context.VolumeExecutionContext{"workspace": "/path"},
				},
			},
			expectedError: "no matching volumes found for server 'test-server': [nonexistent]",
		},
		{
			name: "partial match reports not-found volumes",
			args: []string{"test-server", "workspace", "nonexistent"},
			existingServers: map[string]context.ServerExecutionContext{
				"test-server": {
					Name:       "test-server",
					Volumes:    context.VolumeExecutionContext{"workspace": "/path", "gdrive": "/mcp/gdrive"},
					RawVolumes: context.VolumeExecutionContext{"workspace": "/path", "gdrive": "/mcp/gdrive"},
				},
			},
			expectedOutput:  "Not found (skipped): [nonexistent]",
			expectedVolumes: context.VolumeExecutionContext{"gdrive": "/mcp/gdrive"},
		},
		{
			name: "duplicate volume names are deduplicated",
			args: []string{"test-server", "workspace", "workspace"},
			existingServers: map[string]context.ServerExecutionContext{
				"test-server": {
					Name:       "test-server",
					Volumes:    context.VolumeExecutionContext{"workspace": "/path", "gdrive": "/mcp/gdrive"},
					RawVolumes: context.VolumeExecutionContext{"workspace": "/path", "gdrive": "/mcp/gdrive"},
				},
			},
			expectedOutput:  "✓ Volumes removed for server 'test-server' (operation: updated): [workspace]",
			expectedVolumes: context.VolumeExecutionContext{"gdrive": "/mcp/gdrive"},
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

			var output bytes.Buffer
			removeCmd.SetOut(&output)
			removeCmd.SetErr(&output)

			removeCmd.SetArgs(tc.args)
			err = removeCmd.Execute()

			if tc.expectedError != "" {
				require.EqualError(t, err, tc.expectedError)
				return
			}

			require.NoError(t, err)

			actualOutput := strings.TrimSpace(output.String())
			require.Contains(t, actualOutput, tc.expectedOutput)

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
			name:          "missing all arguments",
			args:          []string{},
			expectedError: "requires at least 2 arg(s), only received 0",
		},
		{
			name:          "missing volume names",
			args:          []string{"server"},
			expectedError: "requires at least 2 arg(s), only received 1",
		},
		{
			name:          "whitespace-only volume name",
			args:          []string{"server", "   "},
			expectedError: "volume name argument cannot be empty",
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
			require.ErrorContains(t, err, tc.expectedError)
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

	removeCmd.SetArgs([]string{"server", "workspace"})
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

	removeCmd.SetArgs([]string{"server", "workspace"})
	err = removeCmd.Execute()
	require.EqualError(t, err, "error removing volumes for server 'server': upsert failed")
}
