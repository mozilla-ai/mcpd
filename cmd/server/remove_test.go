package server

import (
	"bytes"
	"os"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd-cli/v2/internal/config"
	"github.com/mozilla-ai/mcpd-cli/v2/internal/flags"
)

func TestRemoveServer(t *testing.T) {
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
			name:               "basic server remove",
			args:               []string{"first-server"},
			expectedNumServers: 0,
			expectedOutputs: []string{
				"✓ Removed server 'first-server'",
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
			expectedNumServers: 0,
			expectedOutputs: []string{
				"✓ Removed server 'test-server-with-spaces'",
			},
			setupFn: func(t *testing.T, configPath string) {
				// Create a config file with an existing server
				initialContent := `[[servers]]
name = "test-server-with-spaces"
package = "modelcontextprotocol/test-server-with-spaces@latest"
`
				err := os.WriteFile(configPath, []byte(initialContent), 0o644)
				require.NoError(t, err)
			},
		},
		{
			name:               "existing config file should leave others",
			args:               []string{"second-server"},
			expectedNumServers: 1,
			expectedOutputs: []string{
				"✓ Removed server 'second-server'",
			},
			setupFn: func(t *testing.T, configPath string) {
				// Create a config file with an existing server
				initialContent := `[[servers]]
name = "first-server"
package = "modelcontextprotocol/first-server@latest"
				
[[servers]]
name = "second-server"
package = "modelcontextprotocol/second-server@latest"
`
				err := os.WriteFile(configPath, []byte(initialContent), 0o644)
				require.NoError(t, err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tempFile, err := os.CreateTemp(tmpDir, ".mcpd.toml")
			require.NoError(t, err)

			if tc.setupFn != nil {
				tc.setupFn(t, tempFile.Name())
			}

			// Create a buffer to capture output
			output := &bytes.Buffer{}

			// Create a test logger that won't output during tests
			logger := hclog.New(&hclog.LoggerOptions{
				Name:   "test",
				Level:  hclog.Debug,
				Output: output,
			})

			// Create the command
			c := NewRemoveCmd(logger)
			c.SetOut(output)
			c.SetErr(output)
			c.SetArgs(tc.args)

			// Temporarily modify the config file flag value.
			previousConfigFile := flags.ConfigFile
			defer func() { flags.ConfigFile = previousConfigFile }()
			flags.ConfigFile = tempFile.Name()

			// Execute the command
			err = c.Execute()

			// Check error expectations
			if tc.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
				return
			}

			// No error expected
			assert.NoError(t, err)

			// Check output expectations
			outputStr := output.String()
			for _, expectedOutput := range tc.expectedOutputs {
				assert.Contains(t, outputStr, expectedOutput)
			}

			var parsed config.Config
			_, err = toml.DecodeFile(tempFile.Name(), &parsed)
			require.NoError(t, err)
			require.Len(t, parsed.Servers, tc.expectedNumServers)
		})
	}
}
