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

func TestNewListCmd(t *testing.T) {
	t.Parallel()

	base := &cmd.BaseCmd{}
	c, err := NewListCmd(base)
	require.NoError(t, err)
	require.NotNil(t, c)

	require.Equal(t, "list <server-name>", c.Use)
	require.Contains(t, c.Short, "List configured volume mappings")
	require.NotNil(t, c.RunE)
}

func TestListCmd_run(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		serverName      string
		existingServers map[string]context.ServerExecutionContext
		expectedOutput  string
		expectedError   string
	}{
		{
			name:       "list volumes for server with volumes",
			serverName: "filesystem",
			existingServers: map[string]context.ServerExecutionContext{
				"filesystem": {
					Name: "filesystem",
					Volumes: context.VolumeExecutionContext{
						"workspace": "/Users/foo/repos/mcpd",
					},
				},
			},
			expectedOutput: "Volumes for 'filesystem':\n  workspace = /Users/foo/repos/mcpd",
		},
		{
			name:       "list volumes for server with multiple volumes sorted",
			serverName: "filesystem",
			existingServers: map[string]context.ServerExecutionContext{
				"filesystem": {
					Name: "filesystem",
					Volumes: context.VolumeExecutionContext{
						"workspace": "/Users/foo/repos",
						"data":      "my-named-volume",
						"gdrive":    "/mcp/gdrive",
					},
				},
			},
			expectedOutput: "Volumes for 'filesystem':\n  data = my-named-volume\n  gdrive = /mcp/gdrive\n  workspace = /Users/foo/repos",
		},
		{
			name:       "list volumes for server with no volumes",
			serverName: "filesystem",
			existingServers: map[string]context.ServerExecutionContext{
				"filesystem": {
					Name:    "filesystem",
					Volumes: context.VolumeExecutionContext{},
				},
			},
			expectedOutput: "Volumes for 'filesystem':\n  (No volumes set)",
		},
		{
			name:       "list volumes for server with nil volumes",
			serverName: "filesystem",
			existingServers: map[string]context.ServerExecutionContext{
				"filesystem": {
					Name: "filesystem",
				},
			},
			expectedOutput: "Volumes for 'filesystem':\n  (No volumes set)",
		},
		{
			name:            "server not found",
			serverName:      "nonexistent",
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
			listCmd, err := NewListCmd(base, cmdopts.WithContextLoader(loader))
			require.NoError(t, err)

			// Capture output.
			var output bytes.Buffer
			listCmd.SetOut(&output)
			listCmd.SetErr(&output)

			listCmd.SetArgs([]string{tc.serverName})

			err = listCmd.Execute()

			if tc.expectedError != "" {
				require.EqualError(t, err, tc.expectedError)
				return
			}

			require.NoError(t, err)

			actualOutput := strings.TrimSpace(output.String())
			require.Equal(t, tc.expectedOutput, actualOutput)
		})
	}
}

func TestListCmd_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		expectedError string
	}{
		{
			name:          "missing server name",
			args:          []string{},
			expectedError: "accepts 1 arg(s), received 0",
		},
		{
			name:          "empty server name",
			args:          []string{""},
			expectedError: "server-name is required",
		},
		{
			name:          "whitespace only server name",
			args:          []string{"   "},
			expectedError: "server-name is required",
		},
		{
			name:          "too many args",
			args:          []string{"server1", "server2"},
			expectedError: "accepts 1 arg(s), received 2",
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
			listCmd, err := NewListCmd(base, cmdopts.WithContextLoader(loader))
			require.NoError(t, err)

			listCmd.SetArgs(tc.args)
			err = listCmd.Execute()
			require.EqualError(t, err, tc.expectedError)
		})
	}
}

func TestListCmd_LoaderError(t *testing.T) {
	t.Parallel()

	loader := &mockLoader{
		loadError: fmt.Errorf("failed to load"),
	}

	base := &cmd.BaseCmd{}
	listCmd, err := NewListCmd(base, cmdopts.WithContextLoader(loader))
	require.NoError(t, err)

	listCmd.SetArgs([]string{"server"})
	err = listCmd.Execute()
	require.EqualError(t, err, "failed to load execution context config: failed to load")
}
