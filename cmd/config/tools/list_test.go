package tools

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	internalcmd "github.com/mozilla-ai/mcpd/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd/v2/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
)

type mockConfigLoader struct {
	servers []config.ServerEntry
	err     error
}

func (m *mockConfigLoader) Load(path string) (config.Modifier, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &mockConfig{servers: m.servers}, nil
}

type mockConfig struct {
	servers []config.ServerEntry
}

func (m *mockConfig) AddServer(entry config.ServerEntry) error {
	return nil
}

func (m *mockConfig) RemoveServer(name string) error {
	return nil
}

func (m *mockConfig) ListServers() []config.ServerEntry {
	return m.servers
}

func (m *mockConfig) SaveConfig() error {
	return nil
}

func TestListCmd_Run(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		args           []string
		format         string
		servers        []config.ServerEntry
		loaderErr      error
		expectedOutput string
		expectedError  string
	}{
		{
			name:   "server with tools - text format",
			args:   []string{"test-server"},
			format: "text",
			servers: []config.ServerEntry{
				{
					Name:    "test-server",
					Package: "uvx::test-server@1.0.0",
					Tools:   []string{"tool_b", "tool_a", "tool_c"},
				},
			},
			expectedOutput: "Tools for 'test-server' (3 total):\n  tool_a\n  tool_b\n  tool_c\n",
		},
		{
			name:   "server with no tools - text format",
			args:   []string{"empty-server"},
			format: "text",
			servers: []config.ServerEntry{
				{
					Name:    "empty-server",
					Package: "uvx::empty-server@1.0.0",
					Tools:   []string{},
				},
			},
			expectedOutput: "Tools for 'empty-server' (0 total):\n  (No tools configured)\n",
		},
		{
			name:   "server with tools - json format",
			args:   []string{"json-server"},
			format: "json",
			servers: []config.ServerEntry{
				{
					Name:    "json-server",
					Package: "uvx::json-server@1.0.0",
					Tools:   []string{"tool_2", "tool_1"},
				},
			},
			expectedOutput: `{
  "result": {
    "server": "json-server",
    "tools": [
      "tool_1",
      "tool_2"
    ],
    "count": 2
  }
}
`,
		},
		{
			name:          "server not found",
			args:          []string{"nonexistent"},
			format:        "text",
			servers:       []config.ServerEntry{},
			expectedError: "server 'nonexistent' not found in configuration",
		},
		{
			name:          "empty server name",
			args:          []string{"  "},
			format:        "text",
			servers:       []config.ServerEntry{},
			expectedError: "server-name is required",
		},
		{
			name:          "config load error",
			args:          []string{"any-server"},
			format:        "text",
			loaderErr:     errors.New("config load failed"),
			expectedError: "config load failed",
		},
		{
			name:   "multiple servers - finds correct one",
			args:   []string{"server-2"},
			format: "text",
			servers: []config.ServerEntry{
				{
					Name:    "server-1",
					Package: "uvx::server-1@1.0.0",
					Tools:   []string{"tool_1"},
				},
				{
					Name:    "server-2",
					Package: "uvx::server-2@1.0.0",
					Tools:   []string{"tool_2"},
				},
				{
					Name:    "server-3",
					Package: "uvx::server-3@1.0.0",
					Tools:   []string{"tool_3"},
				},
			},
			expectedOutput: "Tools for 'server-2' (1 total):\n  tool_2\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			// Create mock loader
			loader := &mockConfigLoader{
				servers: tc.servers,
				err:     tc.loaderErr,
			}

			// Create base command
			baseCmd := &internalcmd.BaseCmd{}

			// Create the list command with mock loader
			cmd, err := NewListCmd(baseCmd, options.WithConfigLoader(loader))
			require.NoError(t, err)

			// Set output buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			// Set args and format
			args := tc.args
			if tc.format != "" && tc.format != "text" {
				args = append(args, "--format", tc.format)
			}
			cmd.SetArgs(args)

			// Execute command
			err = cmd.Execute()

			// Check error
			if tc.expectedError != "" {
				require.Error(t, err)
				require.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expectedOutput, buf.String())
			}
		})
	}
}

func TestListCmd_Formats(t *testing.T) {
	t.Parallel()

	server := config.ServerEntry{
		Name:    "format-test-server",
		Package: "uvx::format-test@1.0.0",
		Tools:   []string{"create", "read", "update"},
	}

	tests := []struct {
		name           string
		format         string
		expectedOutput string
	}{
		{
			name:           "text format",
			format:         "text",
			expectedOutput: "Tools for 'format-test-server' (3 total):\n  create\n  read\n  update\n",
		},
		{
			name:   "json format",
			format: "json",
			expectedOutput: `{
  "result": {
    "server": "format-test-server",
    "tools": [
      "create",
      "read",
      "update"
    ],
    "count": 3
  }
}
`,
		},
		{
			name:   "yaml format",
			format: "yaml",
			expectedOutput: `result:
  server: format-test-server
  tools:
    - create
    - read
    - update
  count: 3
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer

			loader := &mockConfigLoader{
				servers: []config.ServerEntry{server},
			}

			baseCmd := &internalcmd.BaseCmd{}
			cmd, err := NewListCmd(baseCmd, options.WithConfigLoader(loader))
			require.NoError(t, err)

			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs([]string{"format-test-server", "--format", tc.format})

			err = cmd.Execute()
			require.NoError(t, err)

			require.Equal(t, tc.expectedOutput, buf.String())
		})
	}
}

func TestNewListCmd(t *testing.T) {
	t.Parallel()

	t.Run("creates command successfully", func(t *testing.T) {
		t.Parallel()

		baseCmd := &internalcmd.BaseCmd{}
		cmd, err := NewListCmd(baseCmd)

		require.NoError(t, err)
		require.NotNil(t, cmd)

		require.Equal(t, "list <server-name>", cmd.Use)
		require.Equal(t, "Lists the configured tools for a specific MCP server", cmd.Short)
		require.NotNil(t, cmd.RunE)

		// Check format flag exists
		flag := cmd.Flag("format")
		require.NotNil(t, flag)
		require.Equal(t, "format", flag.Name)
	})

	t.Run("validates exactly one argument", func(t *testing.T) {
		t.Parallel()

		baseCmd := &internalcmd.BaseCmd{}
		cmd, err := NewListCmd(baseCmd)
		require.NoError(t, err)

		// Test with no args
		err = cmd.Args(cmd, []string{})
		require.Error(t, err)

		// Test with one arg
		err = cmd.Args(cmd, []string{"server"})
		require.NoError(t, err)

		// Test with two args
		err = cmd.Args(cmd, []string{"server1", "server2"})
		require.Error(t, err)
	})
}
