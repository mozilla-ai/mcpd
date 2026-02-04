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

// mockModifier implements context.Modifier for testing.
type mockModifier struct {
	servers     map[string]context.ServerExecutionContext
	upsertError error
	lastUpsert  context.ServerExecutionContext
}

func (m *mockModifier) Get(name string) (context.ServerExecutionContext, bool) {
	server, ok := m.servers[name]
	if ok {
		server.Name = name
		return server, true
	}
	return context.ServerExecutionContext{}, false
}

func (m *mockModifier) List() []context.ServerExecutionContext {
	servers := make([]context.ServerExecutionContext, 0, len(m.servers))
	for name, server := range m.servers {
		server.Name = name
		servers = append(servers, server)
	}
	return servers
}

func (m *mockModifier) Upsert(ec context.ServerExecutionContext) (context.UpsertResult, error) {
	m.lastUpsert = ec
	if m.upsertError != nil {
		return context.Noop, m.upsertError
	}
	if _, exists := m.servers[ec.Name]; exists {
		m.servers[ec.Name] = ec
		return context.Updated, nil
	}
	m.servers[ec.Name] = ec
	return context.Created, nil
}

// mockLoader implements context.Loader for testing.
type mockLoader struct {
	modifier  *mockModifier
	loadError error
}

func (m *mockLoader) Load(_ string) (context.Modifier, error) {
	if m.loadError != nil {
		return nil, m.loadError
	}
	return m.modifier, nil
}

func TestNewSetCmd(t *testing.T) {
	t.Parallel()

	base := &cmd.BaseCmd{}
	c, err := NewSetCmd(base)
	require.NoError(t, err)
	require.NotNil(t, c)

	require.Equal(t, "set", c.Use[:3])
	require.Contains(t, c.Short, "Set or update volume mappings")
	require.NotNil(t, c.RunE)
}

func TestSetCmd_run(t *testing.T) {
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
			name:            "add volume to new server",
			serverName:      "test-server",
			volumeArgs:      []string{"--workspace=/Users/foo/repos"},
			existingServers: map[string]context.ServerExecutionContext{},
			expectedOutput:  "✓ Volumes set for server 'test-server' (operation: created): [workspace]",
			expectedVolumes: context.VolumeExecutionContext{"workspace": "/Users/foo/repos"},
		},
		{
			name:       "add volume to existing server",
			serverName: "test-server",
			volumeArgs: []string{"--gdrive=/mcp/gdrive"},
			existingServers: map[string]context.ServerExecutionContext{
				"test-server": {
					Name:    "test-server",
					Volumes: context.VolumeExecutionContext{"workspace": "/existing/path"},
				},
			},
			expectedOutput:  "✓ Volumes set for server 'test-server' (operation: updated): [gdrive]",
			expectedVolumes: context.VolumeExecutionContext{"workspace": "/existing/path", "gdrive": "/mcp/gdrive"},
		},
		{
			name:       "update existing volume",
			serverName: "test-server",
			volumeArgs: []string{"--workspace=/new/path"},
			existingServers: map[string]context.ServerExecutionContext{
				"test-server": {
					Name:    "test-server",
					Volumes: context.VolumeExecutionContext{"workspace": "/old/path"},
				},
			},
			expectedOutput:  "✓ Volumes set for server 'test-server' (operation: updated): [workspace]",
			expectedVolumes: context.VolumeExecutionContext{"workspace": "/new/path"},
		},
		{
			name:            "add multiple volumes",
			serverName:      "test-server",
			volumeArgs:      []string{"--workspace=/Users/foo/repos", "--gdrive=/mcp/gdrive"},
			existingServers: map[string]context.ServerExecutionContext{},
			expectedOutput:  "✓ Volumes set for server 'test-server' (operation: created):",
			expectedVolumes: context.VolumeExecutionContext{"workspace": "/Users/foo/repos", "gdrive": "/mcp/gdrive"},
		},
		{
			name:            "volume with quoted path",
			serverName:      "test-server",
			volumeArgs:      []string{`--workspace="/Users/foo/my repos"`},
			existingServers: map[string]context.ServerExecutionContext{},
			expectedOutput:  "✓ Volumes set for server 'test-server' (operation: created): [workspace]",
			expectedVolumes: context.VolumeExecutionContext{"workspace": "/Users/foo/my repos"},
		},
		{
			name:            "named docker volume",
			serverName:      "test-server",
			volumeArgs:      []string{"--data=my-named-volume"},
			existingServers: map[string]context.ServerExecutionContext{},
			expectedOutput:  "✓ Volumes set for server 'test-server' (operation: created): [data]",
			expectedVolumes: context.VolumeExecutionContext{"data": "my-named-volume"},
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
			setCmd, err := NewSetCmd(base, cmdopts.WithContextLoader(loader))
			require.NoError(t, err)

			// Capture output.
			var output bytes.Buffer
			setCmd.SetOut(&output)
			setCmd.SetErr(&output)

			// Build args with -- separator.
			allArgs := append([]string{tc.serverName, "--"}, tc.volumeArgs...)
			setCmd.SetArgs(allArgs)

			err = setCmd.Execute()

			if tc.expectedError != "" {
				require.EqualError(t, err, tc.expectedError)
				return
			}

			require.NoError(t, err)

			actualOutput := strings.TrimSpace(output.String())
			require.Contains(t, actualOutput, tc.expectedOutput)

			// Verify the volumes were set correctly.
			if tc.expectedVolumes != nil {
				require.Equal(t, tc.expectedVolumes, modifier.lastUpsert.Volumes)
			}
		})
	}
}

func TestSetCmd_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		expectedError string
	}{
		{
			name:          "missing server name",
			args:          []string{"--", "--workspace=/path"},
			expectedError: "server-name is required",
		},
		{
			name:          "empty server name",
			args:          []string{"", "--", "--workspace=/path"},
			expectedError: "server-name is required",
		},
		{
			name:          "no volume mappings",
			args:          []string{"server", "--"},
			expectedError: "volume mapping(s) required after --",
		},
		{
			name:          "missing -- separator",
			args:          []string{"server"},
			expectedError: "missing '--' separator: usage: mcpd config volumes set <server-name> -- --<volume>=<path>",
		},
		{
			name:          "invalid volume format - no prefix",
			args:          []string{"server", "--", "workspace=/path"},
			expectedError: "invalid volume format 'workspace=/path': must start with --",
		},
		{
			name:          "invalid volume format - no equals",
			args:          []string{"server", "--", "--workspace"},
			expectedError: "invalid volume format '--workspace': expected --<volume-name>=<host-path>",
		},
		{
			name:          "empty volume name",
			args:          []string{"server", "--", "--=/path"},
			expectedError: "volume name cannot be empty in '--=/path'",
		},
		{
			name:          "empty volume path",
			args:          []string{"server", "--", "--workspace="},
			expectedError: "volume path cannot be empty for volume 'workspace'",
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
			setCmd, err := NewSetCmd(base, cmdopts.WithContextLoader(loader))
			require.NoError(t, err)

			setCmd.SetArgs(tc.args)
			err = setCmd.Execute()
			require.EqualError(t, err, tc.expectedError)
		})
	}
}

func TestSetCmd_LoaderError(t *testing.T) {
	t.Parallel()

	loader := &mockLoader{
		loadError: fmt.Errorf("failed to load"),
	}

	base := &cmd.BaseCmd{}
	setCmd, err := NewSetCmd(base, cmdopts.WithContextLoader(loader))
	require.NoError(t, err)

	setCmd.SetArgs([]string{"server", "--", "--workspace=/path"})
	err = setCmd.Execute()
	require.EqualError(t, err, "failed to load execution context config: failed to load")
}

func TestSetCmd_UpsertError(t *testing.T) {
	t.Parallel()

	modifier := &mockModifier{
		servers:     map[string]context.ServerExecutionContext{},
		upsertError: fmt.Errorf("upsert failed"),
	}
	loader := &mockLoader{modifier: modifier}

	base := &cmd.BaseCmd{}
	setCmd, err := NewSetCmd(base, cmdopts.WithContextLoader(loader))
	require.NoError(t, err)

	setCmd.SetArgs([]string{"server", "--", "--workspace=/path"})
	err = setCmd.Execute()
	require.EqualError(t, err, "error setting volumes for server 'server': upsert failed")
}

func TestValidateArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		dashPos       int
		expectedError string
	}{
		{
			name:          "valid args",
			args:          []string{"server", "--workspace=/path"},
			dashPos:       1,
			expectedError: "",
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
			expectedError: "missing '--' separator: usage: mcpd config volumes set <server-name> -- --<volume>=<path>",
		},
		{
			name:          "empty server name at position 0",
			args:          []string{"", "--workspace=/path"},
			dashPos:       1,
			expectedError: "server-name is required",
		},
		{
			name:          "whitespace only server name",
			args:          []string{"   ", "--workspace=/path"},
			dashPos:       1,
			expectedError: "server-name is required",
		},
		{
			name:          "dash at position 0",
			args:          []string{"--workspace=/path"},
			dashPos:       0,
			expectedError: "server-name is required",
		},
		{
			name:          "too many args before dash",
			args:          []string{"server", "extra", "--workspace=/path"},
			dashPos:       2,
			expectedError: "too many arguments before --",
		},
		{
			name:          "no volume mappings after dash",
			args:          []string{"server"},
			dashPos:       1,
			expectedError: "volume mapping(s) required after --",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := validateArgs(tc.dashPos, tc.args)

			if tc.expectedError != "" {
				require.EqualError(t, err, tc.expectedError)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestParseVolumeArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		expected      map[string]string
		expectedError string
	}{
		{
			name:     "single volume",
			args:     []string{"--workspace=/path/to/workspace"},
			expected: map[string]string{"workspace": "/path/to/workspace"},
		},
		{
			name:     "multiple volumes",
			args:     []string{"--workspace=/path1", "--gdrive=/path2"},
			expected: map[string]string{"workspace": "/path1", "gdrive": "/path2"},
		},
		{
			name:     "volume with double quotes",
			args:     []string{`--workspace="/path with spaces"`},
			expected: map[string]string{"workspace": "/path with spaces"},
		},
		{
			name:     "volume with single quotes",
			args:     []string{`--workspace='/path with spaces'`},
			expected: map[string]string{"workspace": "/path with spaces"},
		},
		{
			name:          "missing -- prefix",
			args:          []string{"workspace=/path"},
			expectedError: "invalid volume format 'workspace=/path': must start with --",
		},
		{
			name:          "missing =",
			args:          []string{"--workspace"},
			expectedError: "invalid volume format '--workspace': expected --<volume-name>=<host-path>",
		},
		{
			name:          "empty name",
			args:          []string{"--=/path"},
			expectedError: "volume name cannot be empty in '--=/path'",
		},
		{
			name:          "empty path",
			args:          []string{"--workspace="},
			expectedError: "volume path cannot be empty for volume 'workspace'",
		},
		{
			name:          "empty quoted path",
			args:          []string{`--data=""`},
			expectedError: "volume path cannot be empty for volume 'data'",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := parseVolumeArgs(tc.args)

			if tc.expectedError != "" {
				require.EqualError(t, err, tc.expectedError)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestTrimQuotes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no quotes",
			input:    "/path/to/file",
			expected: "/path/to/file",
		},
		{
			name:     "double quotes",
			input:    `"/path/to/file"`,
			expected: "/path/to/file",
		},
		{
			name:     "single quotes",
			input:    "'/path/to/file'",
			expected: "/path/to/file",
		},
		{
			name:     "mismatched quotes - double then single",
			input:    `"/path/to/file'`,
			expected: `"/path/to/file'`,
		},
		{
			name:     "mismatched quotes - single then double",
			input:    `'/path/to/file"`,
			expected: `'/path/to/file"`,
		},
		{
			name:     "single character",
			input:    "a",
			expected: "a",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := trimQuotes(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}
