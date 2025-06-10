package server

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd-cli/v2/internal/cmd"
	"github.com/mozilla-ai/mcpd-cli/v2/internal/config"
	"github.com/mozilla-ai/mcpd-cli/v2/internal/discover"
	"github.com/mozilla-ai/mcpd-cli/v2/internal/flags"
)

// MockDiscoverer implements the discover.Discoverer interface for testing
type MockDiscoverer struct{}

// DiscoverPackage is a mock implementation that always succeeds
func (m *MockDiscoverer) DiscoverPackage(packageName, version string) (discover.DiscoveryResult, error) {
	return discover.DiscoveryResult{
		Version:          version,
		FoundTools:       []string{"tool1", "tool2", "tool3"},
		FoundStartupArgs: []string{"--port", "--host"},
		FoundEnvVars:     true,
	}, nil
}

func TestAddCmd_Execute(t *testing.T) {
	tests := []struct {
		name               string
		args               []string
		expectedNumServers int
		expectedVersion    string
		expectedTools      []string
		expectedOutputs    []string
		expectedError      string
		setupFn            func(t *testing.T, configPath string) // Optional setup function
	}{
		{
			name:               "basic server add",
			args:               []string{"testserver"},
			expectedNumServers: 1,
			expectedOutputs: []string{
				"✓ Added server 'testserver'",
				"version: latest",
			},
		},
		{
			name:               "server add with version",
			args:               []string{"testserver", "--version", "1.2.3"},
			expectedNumServers: 1,
			expectedVersion:    "1.2.3",
			expectedOutputs: []string{
				"✓ Added server 'testserver' (version: 1.2.3)",
			},
		},
		{
			name:               "server add with tools",
			args:               []string{"testserver", "--tool", "tool1", "--tool", "tool2"},
			expectedNumServers: 1,
			expectedTools:      []string{"tool1", "tool2"},
			expectedOutputs: []string{
				"✓ Added server 'testserver' (version: latest), exposing only tools: tool1, tool2",
			},
		},
		{
			name:          "missing server name",
			args:          []string{},
			expectedError: "server name is required and cannot be empty",
		},
		{
			name:          "empty server name",
			args:          []string{"  "},
			expectedError: "server name is required and cannot be empty",
		},
		{
			name:               "server name with whitespace",
			args:               []string{" test-server-with-spaces "},
			expectedNumServers: 1,
			expectedOutputs: []string{
				"✓ Added server 'test-server-with-spaces'",
			},
		},
		{
			name:               "existing config file should append",
			args:               []string{"second-server"},
			expectedNumServers: 2,
			expectedOutputs: []string{
				"✓ Added server 'second-server'",
			},
			setupFn: func(t *testing.T, configPath string) {
				// Create a config file with an existing server
				initialContent := `[[servers]]
name = "first-server"
package = "modelcontextprotocol/first-server@latest"
`
				err := os.WriteFile(configPath, []byte(initialContent), 0o644)
				require.NoError(t, err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a temporary directory for the test
			tempDir := t.TempDir()
			tempFile, err := os.CreateTemp(tempDir, "config.toml")
			require.NoError(t, err)

			// Run any setup function if provided
			if tc.setupFn != nil {
				tc.setupFn(t, tempFile.Name())
			}

			// Create a buffer to capture output
			output := &bytes.Buffer{}
			baseCmd := &cmd.BaseCmd{}

			// Create the command
			c := NewAddCmd(baseCmd, &MockDiscoverer{})
			c.SetOut(output)
			c.SetErr(output)
			c.SetArgs(tc.args)

			// Temporarily modify the config file flag value.
			previousConfigFile := flags.ConfigFile
			defer func() { flags.ConfigFile = previousConfigFile }()
			flags.ConfigFile = tempFile.Name()

			// Execute the command
			err = c.Execute()

			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				return
			}
			assert.NoError(t, err)

			outputStr := output.String()
			for _, expectedOutput := range tc.expectedOutputs {
				assert.Contains(t, outputStr, expectedOutput)
			}

			var parsed config.Config
			_, err = toml.DecodeFile(tempFile.Name(), &parsed)
			require.NoError(t, err)

			require.Len(t, parsed.Servers, tc.expectedNumServers)
			serverName := strings.TrimSpace(tc.args[0])

			// May have >1 server (if we already had config).
			findByName := func(name string) (config.ServerEntry, bool) {
				for _, entry := range parsed.Servers {
					if entry.Name == name {
						return entry, true
					}
				}
				return config.ServerEntry{}, false
			}

			server, exists := findByName(serverName)
			assert.True(t, exists)
			assert.Equal(t, serverName, server.Name)

			version := "latest"
			if tc.expectedVersion != "" {
				version = tc.expectedVersion
			}
			assert.Equal(t, fmt.Sprintf("pypi::mcp-server-%s@%s", serverName, version), server.Package)

			if tc.expectedTools != nil {
				assert.Equal(t, tc.expectedTools, server.Tools)
			} else {
				// We will have *all* the tools now.
				assert.NotEmpty(t, server.Tools)
				assert.ElementsMatch(t, []string{"tool1", "tool2", "tool3"}, server.Tools)
			}
		})
	}
}

func TestAddCmd_WithCustomConfigPath(t *testing.T) {
	tempDir := t.TempDir()
	tempFile, err := os.CreateTemp(tempDir, "custom-config.toml")
	require.NoError(t, err)

	// Create a buffer to capture output
	output := &bytes.Buffer{}

	// Create the command
	baseCmd := &cmd.BaseCmd{}
	c := NewAddCmd(baseCmd, &MockDiscoverer{})
	c.SetOut(output)
	c.SetErr(output)
	c.SetArgs([]string{"custom-server", "--version", "2.0.0"})

	// Temporarily modify the config file flag value.
	previousConfigFile := flags.ConfigFile
	defer func() { flags.ConfigFile = previousConfigFile }()
	flags.ConfigFile = tempFile.Name()

	// Execute the command
	err = c.Execute()
	require.NoError(t, err)

	// Verify output
	outputStr := output.String()
	assert.Contains(t, outputStr, "✓ Added server 'custom-server'")
	assert.Contains(t, outputStr, "version: 2.0.0")

	// Verify the config file was created at the custom path
	assert.FileExists(t, tempFile.Name())

	// Verify content
	var parsed config.Config
	_, err = toml.DecodeFile(tempFile.Name(), &parsed)
	require.NoError(t, err)

	require.Len(t, parsed.Servers, 1)
	server := parsed.Servers[0]
	assert.Equal(t, "custom-server", server.Name)
	assert.Equal(t, "pypi::mcp-server-custom-server@2.0.0", server.Package)
	assert.ElementsMatch(t, []string{"tool1", "tool2", "tool3"}, server.Tools)
}

func TestAddCmd_LongDescription(t *testing.T) {
	t.Parallel()

	c := &AddCmd{
		BaseCmd: &cmd.BaseCmd{},
	}

	description := c.longDescription()
	assert.Contains(t, description, "Adds an MCP server dependency")
	assert.Contains(t, description, "mcpd will search the registry")
}
