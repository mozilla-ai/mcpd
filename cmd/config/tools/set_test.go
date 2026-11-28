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
	"github.com/mozilla-ai/mcpd/internal/domain"
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
	updateError   error
	serverRemoved string
	serverAdded   config.ServerEntry
	serverUpdated config.ServerEntry
}

// AddServer refuses a name another entry already holds under one key, as the real config refuses it.
func (m *mockConfigForSet) AddServer(entry config.ServerEntry) error {
	m.serverAdded = entry
	if m.addError != nil {
		return m.addError
	}

	key := domain.ServerName(entry.Name).Key()
	for _, srv := range m.servers {
		if domain.ServerName(srv.Name).Key() == key {
			return fmt.Errorf("duplicate server name '%s': already declared as '%s'", entry.Name, srv.Name)
		}
	}

	m.servers = append(m.servers, entry)

	return nil
}

// RemoveServer takes out the entry it resolves and reports a name it does not hold, as the real config
// does, so a caller that passes a spelling the configuration does not contain fails here rather than
// silently succeeding.
func (m *mockConfigForSet) RemoveServer(name string) error {
	m.serverRemoved = name
	if m.removeError != nil {
		return m.removeError
	}

	idx := slices.IndexFunc(m.servers, func(srv config.ServerEntry) bool { return srv.Name == name })
	if idx < 0 {
		return fmt.Errorf("server '%s' not found in config", name)
	}

	m.servers = slices.Delete(m.servers, idx, idx+1)

	return nil
}

// UpdateServer replaces the entry it resolves and reports a name it does not hold, as the real config
// does, so a caller that passes a spelling the configuration does not contain fails here rather than
// silently succeeding.
func (m *mockConfigForSet) UpdateServer(entry config.ServerEntry) error {
	m.serverUpdated = entry
	if m.updateError != nil {
		return m.updateError
	}

	idx := slices.IndexFunc(m.servers, func(srv config.ServerEntry) bool { return srv.Name == entry.Name })
	if idx < 0 {
		return fmt.Errorf("server '%s' not found in config", entry.Name)
	}

	m.servers[idx] = entry

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
			// A server name is matched without regard to case, so the update has to reach the entry
			// the lookup found rather than the spelling that was typed.
			name:       "add tools using a server name in a different case",
			serverName: "TEST-SERVER",
			tools:      []string{"new-tool-1"},
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
					},
				},
			},
			expectedOutput: "✓ Tools added for server 'test-server': [new-tool-1]",
			expectedTools:  []string{"existing-tool", "new-tool-1"},
		},
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
			name:       "reject a tool name that differs only by case",
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
			expectedError: "the following tools are not available for server 'test-server': [Tool_One TOOL-TWO]",
		},
		{
			name:       "record a tool name as the registry spells it",
			serverName: "test-server",
			tools:      []string{"Tool_One", "TOOL-TWO"},
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
						{Name: "Tool_One"},
						{Name: "TOOL-TWO"},
					},
				},
			},
			expectedOutput: "✓ Tools added for server 'test-server': [Tool_One TOOL-TWO]",
			expectedTools:  []string{"TOOL-TWO", "Tool_One"},
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
				// The update must name the entry as the configuration stores it, not as the caller
				// spelled it, or it cannot reach the entry the lookup found.
				require.Equal(t, tc.existingServers[0].Name, cfg.serverUpdated.Name)

				// Check the tools are correct and sorted.
				slices.Sort(cfg.serverUpdated.Tools)
				slices.Sort(tc.expectedTools)
				require.Equal(t, tc.expectedTools, cfg.serverUpdated.Tools)
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
		updateError   error
		registryError error
		expectedError string
	}{
		{
			name:          "registry resolve error",
			registryError: fmt.Errorf("registry unavailable"),
			expectedError: "failed to get available tools for server 'test-server': failed to resolve server 'test-server': registry unavailable",
		},
		{
			name:          "update server error",
			updateError:   fmt.Errorf("update failed"),
			expectedError: "error updating server configuration: update failed",
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
				updateError: tc.updateError,
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

// TestSetCmd_KeepsAnEntryWhoseNameSharesOneKey pins that adding a tool to a server in a configuration
// that holds two names sharing one key leaves both entries in place.
func TestSetCmd_KeepsAnEntryWhoseNameSharesOneKey(t *testing.T) {
	t.Parallel()

	cfg := &mockConfigForSet{
		servers: []config.ServerEntry{
			{
				Name:    "GitHub",
				Package: "uvx::official-github@1.0.0",
				Tools:   []string{"search_issues"},
			},
			{
				Name:    "github",
				Package: "uvx::other-github@9.9.9",
				Tools:   []string{"run_command"},
			},
		},
	}

	registryBuilder := &mockRegistryBuilderForSet{
		registry: &mockRegistryForSet{
			servers: map[string]packages.Server{
				"GitHub": {
					Name:  "GitHub",
					Tools: []packages.Tool{{Name: "search_issues"}, {Name: "list_repos"}},
				},
			},
		},
	}

	setCmd, err := NewSetCmd(&cmd.BaseCmd{},
		cmdopts.WithConfigLoader(&mockLoaderForSet{cfg: cfg}),
		cmdopts.WithRegistryBuilder(registryBuilder),
	)
	require.NoError(t, err)
	require.NoError(t, setCmd.Flags().Set("tool", "list_repos"))

	var output bytes.Buffer
	setCmd.SetOut(&output)
	setCmd.SetErr(&output)

	err = setCmd.RunE(setCmd, []string{"GitHub"})
	require.NoError(t, err)

	names := make([]string, 0, len(cfg.servers))
	for _, srv := range cfg.servers {
		names = append(names, srv.Name)
	}
	slices.Sort(names)
	require.Equal(t, []string{"GitHub", "github"}, names)
}
