package tools

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/internal/config"
	"github.com/mozilla-ai/mcpd/internal/packages"
	"github.com/mozilla-ai/mcpd/internal/registry"
	"github.com/mozilla-ai/mcpd/internal/registry/options"
)

// Mock config for testing.
type mockConfigForSet struct {
	servers       []config.ServerEntry
	saveError     error
	removeError   error
	addError      error
	serverRemoved string
	serverAdded   config.ServerEntry
}

func (m *mockConfigForSet) AddServer(entry config.ServerEntry) error {
	m.serverAdded = entry
	if m.addError != nil {
		return m.addError
	}
	// Update the server in our mock list.
	for i, srv := range m.servers {
		if srv.Name == entry.Name {
			m.servers[i] = entry
			return nil
		}
	}
	m.servers = append(m.servers, entry)
	return nil
}

func (m *mockConfigForSet) RemoveServer(name string) error {
	m.serverRemoved = name
	if m.removeError != nil {
		return m.removeError
	}
	// Don't actually remove from list, just track that it was called.
	return nil
}

func (m *mockConfigForSet) ListServers() []config.ServerEntry {
	return m.servers
}

func (m *mockConfigForSet) SaveConfig() error {
	return m.saveError
}

// Mock loader for testing.
type mockLoaderForSet struct {
	cfg *mockConfigForSet
	err error
}

func (m *mockLoaderForSet) Load(_ string) (config.Modifier, error) {
	return m.cfg, m.err
}

// Mock registry for testing.
type mockRegistryForSet struct {
	servers map[string]packages.Server
	err     error
}

func (m *mockRegistryForSet) Resolve(name string, opts ...options.ResolveOption) (packages.Server, error) {
	if m.err != nil {
		return packages.Server{}, m.err
	}
	server, ok := m.servers[name]
	if !ok {
		return packages.Server{}, fmt.Errorf("server not found in registry: %s", name)
	}
	return server, nil
}

func (m *mockRegistryForSet) Search(
	name string,
	filters map[string]string,
	opts ...options.SearchOption,
) ([]packages.Server, error) {
	return nil, fmt.Errorf("search not implemented")
}

func (m *mockRegistryForSet) ID() string {
	return "mock-registry"
}

// Mock registry builder for testing.
type mockRegistryBuilderForSet struct {
	registry *mockRegistryForSet
	err      error
}

func (m *mockRegistryBuilderForSet) Build(opts ...options.BuildOption) (registry.PackageProvider, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.registry, nil
}

func TestNewSetCmd(t *testing.T) {
	t.Parallel()

	base := &cmd.BaseCmd{}
	c, err := NewSetCmd(base)
	require.NoError(t, err)
	require.NotNil(t, c)

	assert.Equal(t, "set", c.Use[:3])
	assert.Contains(t, c.Short, "Add allowed tools")
	assert.NotNil(t, c.RunE)
}

func TestSetCmd_run(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		serverName      string
		tools           []string
		existingServers []config.ServerEntry
		registryServers map[string]packages.Server
		expectedOutput  string
		expectedError   string
		expectedTools   []string
	}{
		{
			name:       "add new tools to server with existing tools",
			serverName: "test-server",
			tools:      []string{"new-tool-1", "new-tool-2"},
			existingServers: []config.ServerEntry{
				{
					Name:    "test-server",
					Package: "github.com/example/test-server@npx",
					Tools:   []string{"existing-tool"},
				},
			},
			registryServers: map[string]packages.Server{
				"test-server": {
					Name: "test-server",
					Tools: []packages.Tool{
						{Name: "existing-tool"},
						{Name: "new-tool-1"},
						{Name: "new-tool-2"},
						{Name: "other-tool"},
					},
				},
			},
			expectedOutput: "✓ Tools added for server 'test-server': [new-tool-1 new-tool-2]",
			expectedTools:  []string{"existing-tool", "new-tool-1", "new-tool-2"},
		},
		{
			name:       "add tools to server with no existing tools",
			serverName: "test-server",
			tools:      []string{"tool-1", "tool-2"},
			existingServers: []config.ServerEntry{
				{
					Name:    "test-server",
					Package: "github.com/example/test-server@npx",
				},
			},
			registryServers: map[string]packages.Server{
				"test-server": {
					Name: "test-server",
					Tools: []packages.Tool{
						{Name: "tool-1"},
						{Name: "tool-2"},
						{Name: "tool-3"},
					},
				},
			},
			expectedOutput: "✓ Tools added for server 'test-server': [tool-1 tool-2]",
			expectedTools:  []string{"tool-1", "tool-2"},
		},
		{
			name:       "add duplicate tools (idempotent)",
			serverName: "test-server",
			tools:      []string{"existing-tool", "new-tool"},
			existingServers: []config.ServerEntry{
				{
					Name:    "test-server",
					Package: "github.com/example/test-server@npx",
					Tools:   []string{"existing-tool"},
				},
			},
			registryServers: map[string]packages.Server{
				"test-server": {
					Name: "test-server",
					Tools: []packages.Tool{
						{Name: "existing-tool"},
						{Name: "new-tool"},
					},
				},
			},
			expectedOutput: "✓ Tools added for server 'test-server': [new-tool]",
			expectedTools:  []string{"existing-tool", "new-tool"},
		},
		{
			name:       "all tools already exist",
			serverName: "test-server",
			tools:      []string{"tool-1", "tool-2"},
			existingServers: []config.ServerEntry{
				{
					Name:    "test-server",
					Package: "github.com/example/test-server@npx",
					Tools:   []string{"tool-1", "tool-2", "tool-3"},
				},
			},
			registryServers: map[string]packages.Server{
				"test-server": {
					Name: "test-server",
					Tools: []packages.Tool{
						{Name: "tool-1"},
						{Name: "tool-2"},
						{Name: "tool-3"},
					},
				},
			},
			expectedOutput: "✓ No new tools added for server 'test-server' (all specified tools already exist)",
			expectedTools:  []string{"tool-1", "tool-2", "tool-3"},
		},
		{
			name:       "normalize tool names",
			serverName: "test-server",
			tools:      []string{"Tool_One", "TOOL-TWO", "tool three"},
			existingServers: []config.ServerEntry{
				{
					Name:    "test-server",
					Package: "github.com/example/test-server@npx",
				},
			},
			registryServers: map[string]packages.Server{
				"test-server": {
					Name: "test-server",
					Tools: []packages.Tool{
						{Name: "tool_one"},
						{Name: "tool-two"},
						{Name: "tool three"},
					},
				},
			},
			expectedOutput: "✓ Tools added for server 'test-server': [tool_one tool-two tool three]",
			expectedTools:  []string{"tool_one", "tool-two", "tool three"},
		},
		{
			name:       "server not found",
			serverName: "non-existent",
			tools:      []string{"tool-1"},
			existingServers: []config.ServerEntry{
				{
					Name:    "test-server",
					Package: "github.com/example/test-server@npx",
				},
			},
			registryServers: map[string]packages.Server{
				"test-server": {
					Name: "test-server",
					Tools: []packages.Tool{
						{Name: "tool-1"},
					},
				},
			},
			expectedError: "server 'non-existent' not found in configuration",
		},
		{
			name:       "tool not available in registry",
			serverName: "test-server",
			tools:      []string{"invalid-tool"},
			existingServers: []config.ServerEntry{
				{
					Name:    "test-server",
					Package: "github.com/example/test-server@npx",
				},
			},
			registryServers: map[string]packages.Server{
				"test-server": {
					Name: "test-server",
					Tools: []packages.Tool{
						{Name: "valid-tool"},
					},
				},
			},
			expectedError: "the following tools are not available for server 'test-server': [invalid-tool]",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Create mock config and loader.
			cfg := &mockConfigForSet{
				servers: tc.existingServers,
			}
			loader := &mockLoaderForSet{cfg: cfg}

			// Create mock registry and builder.
			registryBuilder := &mockRegistryBuilderForSet{
				registry: &mockRegistryForSet{
					servers: tc.registryServers,
				},
			}

			base := &cmd.BaseCmd{}
			setCmd, err := NewSetCmd(base,
				cmdopts.WithConfigLoader(loader),
				cmdopts.WithRegistryBuilder(registryBuilder),
			)
			require.NoError(t, err)

			// Set the tools via flags.
			for _, tool := range tc.tools {
				err = setCmd.Flags().Set("tool", tool)
				require.NoError(t, err)
			}

			// Capture output.
			var output bytes.Buffer
			setCmd.SetOut(&output)
			setCmd.SetErr(&output)

			// Run the command.
			err = setCmd.RunE(setCmd, []string{tc.serverName})

			// Check error.
			if tc.expectedError != "" {
				require.EqualError(t, err, tc.expectedError)
				return
			}

			require.NoError(t, err)

			// Check output.
			actualOutput := strings.TrimSpace(output.String())
			require.Contains(t, actualOutput, tc.expectedOutput)

			// Verify the configuration was updated correctly.
			if tc.expectedTools != nil {
				// Check that remove and add were called.
				require.Equal(t, tc.serverName, cfg.serverRemoved)
				require.Equal(t, tc.serverName, cfg.serverAdded.Name)

				// Check the tools are correct and sorted.
				slices.Sort(cfg.serverAdded.Tools)
				slices.Sort(tc.expectedTools)
				require.Equal(t, tc.expectedTools, cfg.serverAdded.Tools)
			}
		})
	}
}

func TestSetCmd_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		setupFlags    func(*cobra.Command)
		expectedError string
	}{
		{
			name:          "missing server name",
			args:          []string{},
			setupFlags:    func(cmd *cobra.Command) { _ = cmd.Flags().Set("tool", "tool-1") },
			expectedError: "accepts 1 arg(s), received 0",
		},
		{
			name:          "empty server name",
			args:          []string{""},
			setupFlags:    func(cmd *cobra.Command) { _ = cmd.Flags().Set("tool", "tool-1") },
			expectedError: "server-name is required",
		},
		{
			name:          "no tools provided",
			args:          []string{"server"},
			setupFlags:    func(cmd *cobra.Command) {},
			expectedError: "required flag(s) \"tool\" not set",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Create mock config.
			cfg := &mockConfigForSet{
				servers: []config.ServerEntry{
					{
						Name:    "server",
						Package: "test@npx",
					},
				},
			}
			loader := &mockLoaderForSet{cfg: cfg}

			// Create mock registry.
			registryBuilder := &mockRegistryBuilderForSet{
				registry: &mockRegistryForSet{
					servers: map[string]packages.Server{
						"server": {
							Name: "server",
							Tools: []packages.Tool{
								{
									Name: "tool-1",
								},
							},
						},
					},
				},
			}

			// Create command.
			base := &cmd.BaseCmd{}
			setCmd, err := NewSetCmd(base,
				cmdopts.WithConfigLoader(loader),
				cmdopts.WithRegistryBuilder(registryBuilder),
			)
			require.NoError(t, err)

			// Setup flags as needed.
			tc.setupFlags(setCmd)

			// Run command through Cobra's execution path to get proper validation.
			setCmd.SetArgs(tc.args)
			err = setCmd.Execute()
			require.EqualError(t, err, tc.expectedError)
		})
	}
}

func TestSetCmd_ConfigErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		removeError   error
		addError      error
		saveError     error
		registryError error
		expectedError string
	}{
		{
			name:          "registry resolve error",
			registryError: fmt.Errorf("registry unavailable"),
			expectedError: "failed to get available tools for server 'test-server': failed to resolve server 'test-server': registry unavailable",
		},
		{
			name:          "remove server error",
			removeError:   fmt.Errorf("remove failed"),
			expectedError: "error updating server configuration: remove failed",
		},
		{
			name:          "add server error",
			addError:      fmt.Errorf("add failed"),
			expectedError: "error updating server configuration: add failed",
		},
		{
			name:          "save config error",
			saveError:     fmt.Errorf("save failed"),
			expectedError: "error saving configuration: save failed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Create mock config with error.
			cfg := &mockConfigForSet{
				servers: []config.ServerEntry{
					{
						Name:    "test-server",
						Package: "test@npx",
						Tools:   []string{"existing-tool"},
					},
				},
				removeError: tc.removeError,
				addError:    tc.addError,
				saveError:   tc.saveError,
			}
			loader := &mockLoaderForSet{cfg: cfg}

			// Create mock registry.
			registryBuilder := &mockRegistryBuilderForSet{
				registry: &mockRegistryForSet{
					servers: map[string]packages.Server{
						"test-server": {
							Name: "test-server",
							Tools: []packages.Tool{
								{Name: "existing-tool"},
								{Name: "new-tool"},
							},
						},
					},
					err: tc.registryError,
				},
			}

			// Create command.
			base := &cmd.BaseCmd{}
			setCmd, err := NewSetCmd(base,
				cmdopts.WithConfigLoader(loader),
				cmdopts.WithRegistryBuilder(registryBuilder),
			)
			require.NoError(t, err)
			_ = setCmd.Flags().Set("tool", "new-tool")
			err = setCmd.RunE(setCmd, []string{"test-server"})
			require.EqualError(t, err, tc.expectedError)
		})
	}
}
