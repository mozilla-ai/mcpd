package tools

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/internal/cmd"
	cmdopts "github.com/mozilla-ai/mcpd/internal/cmd/options"
	"github.com/mozilla-ai/mcpd/internal/config"
)

type mockConfigForRemove struct {
	servers   []config.ServerEntry
	addErr    error
	removeErr error
	updateErr error
}

func (m *mockConfigForRemove) AddServer(entry config.ServerEntry) error {
	if m.addErr != nil {
		return m.addErr
	}
	m.servers = append(m.servers, entry)
	return nil
}

func (m *mockConfigForRemove) RemoveServer(name string) error {
	if m.removeErr != nil {
		return m.removeErr
	}
	for i, s := range m.servers {
		if s.Name == name {
			m.servers = append(m.servers[:i], m.servers[i+1:]...)
			return nil
		}
	}
	return nil
}

// UpdateServer replaces the entry it resolves and reports a name it does not hold, as the real config
// does.
func (m *mockConfigForRemove) UpdateServer(entry config.ServerEntry) error {
	if m.updateErr != nil {
		return m.updateErr
	}

	for i, s := range m.servers {
		if s.Name == entry.Name {
			m.servers[i] = entry

			return nil
		}
	}

	return fmt.Errorf("server '%s' not found in config", entry.Name)
}

func (m *mockConfigForRemove) ListServers() []config.ServerEntry {
	return m.servers
}

func (m *mockConfigForRemove) SaveConfig() error {
	return nil
}

type mockLoaderForRemove struct {
	cfg *mockConfigForRemove
	err error
}

func (m *mockLoaderForRemove) Load(_ string) (config.Modifier, error) {
	return m.cfg, m.err
}

func TestRemoveCmd_MatchesAServerNameWithoutRegardToCase(t *testing.T) {
	t.Parallel()

	cfg := &mockConfigForRemove{
		servers: []config.ServerEntry{
			{
				Name:  "test-server",
				Tools: []string{"tool1", "tool2"},
			},
		},
	}

	removeCmd, err := NewRemoveCmd(&cmd.BaseCmd{}, cmdopts.WithConfigLoader(&mockLoaderForRemove{cfg: cfg}))
	require.NoError(t, err)

	var out bytes.Buffer
	removeCmd.SetOut(&out)
	removeCmd.SetErr(&out)

	err = removeCmd.RunE(removeCmd, []string{"Test-Server", "tool1"})
	require.NoError(t, err)

	// The entry the lookup found is the entry rewritten, so the server is not duplicated.
	require.Len(t, cfg.servers, 1)
	require.Equal(t, "test-server", cfg.servers[0].Name)
	require.Equal(t, []string{"tool2"}, cfg.servers[0].Tools)
	require.Contains(t, out.String(), "✓ Tools removed for server 'test-server'")
}

func TestRemoveCmd_RemoveAllTools(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		serverName    string
		initialTools  []string
		toolsToRemove []string
		expectedError string
	}{
		{
			name:          "removing all tools should fail",
			serverName:    "test-server",
			initialTools:  []string{"tool1", "tool2"},
			toolsToRemove: []string{"tool1", "tool2"},
			expectedError: "cannot remove all tools from server 'test-server'\nTo remove the server instead use: mcpd remove test-server",
		},
		{
			name:          "removing last tool should fail",
			serverName:    "test-server",
			initialTools:  []string{"tool1"},
			toolsToRemove: []string{"tool1"},
			expectedError: "cannot remove all tools from server 'test-server'\nTo remove the server instead use: mcpd remove test-server",
		},
		{
			name:          "removing some tools should succeed",
			serverName:    "test-server",
			initialTools:  []string{"tool1", "tool2", "tool3"},
			toolsToRemove: []string{"tool1", "tool2"},
			expectedError: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := &mockConfigForRemove{
				servers: []config.ServerEntry{
					{
						Name:  tc.serverName,
						Tools: tc.initialTools,
					},
				},
			}

			loader := &mockLoaderForRemove{cfg: cfg}
			baseCmd := &cmd.BaseCmd{}

			removeCmd, err := NewRemoveCmd(baseCmd, cmdopts.WithConfigLoader(loader))
			require.NoError(t, err)

			// Prepare arguments: server-name followed by tools to remove.
			args := append([]string{tc.serverName}, tc.toolsToRemove...)

			var out bytes.Buffer
			removeCmd.SetOut(&out)
			removeCmd.SetErr(&out)

			err = removeCmd.RunE(removeCmd, args)

			if tc.expectedError != "" {
				require.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
				// Verify some tools remain.
				require.Greater(t, len(cfg.servers[0].Tools), 0)
			}
		})
	}
}
