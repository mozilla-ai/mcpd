package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd-cli/v2/internal/flags"
)

func TestAddServer(t *testing.T) {
	tests := []struct {
		name              string
		config            *Config
		newEntry          ServerEntry
		isErrorExpected   bool
		expectedErrMsg    string
		shouldSetupConfig bool
	}{
		{
			name: "add server to existing config",
			config: &Config{
				Servers: []ServerEntry{
					{Name: "existing-server", Package: "modelcontextprotocol/existing@v1.0.0"},
				},
			},
			newEntry: ServerEntry{
				Name:    "new-server",
				Package: "modelcontextprotocol/new-server@latest",
				Tools:   []string{"tool1"},
			},
			shouldSetupConfig: true,
			isErrorExpected:   false,
		},
		{
			name:   "add server to empty config",
			config: &Config{Servers: []ServerEntry{}},
			newEntry: ServerEntry{
				Name:    "first-server",
				Package: "modelcontextprotocol/first-server@latest",
			},
			shouldSetupConfig: true,
			isErrorExpected:   false,
		},
		{
			name: "add duplicate server (same name and package base)",
			config: &Config{
				Servers: []ServerEntry{
					{Name: "test-server", Package: "modelcontextprotocol/test-server@v1.0.0"},
				},
			},
			newEntry: ServerEntry{
				Name:    "test-server",
				Package: "modelcontextprotocol/test-server@v2.0.0",
			},
			shouldSetupConfig: true,
			isErrorExpected:   true,
			expectedErrMsg:    "duplicate server entry",
		},
		{
			name:   "add server with empty name",
			config: &Config{Servers: []ServerEntry{}},
			newEntry: ServerEntry{
				Name:    "",
				Package: "modelcontextprotocol/test-server@latest",
			},
			shouldSetupConfig: true,
			isErrorExpected:   true,
			expectedErrMsg:    "server entry has empty name",
		},
		{
			name:   "add server with empty package",
			config: &Config{Servers: []ServerEntry{}},
			newEntry: ServerEntry{
				Name:    "test-server",
				Package: "",
			},
			shouldSetupConfig: true,
			isErrorExpected:   true,
			expectedErrMsg:    "server entry has empty package",
		},
		// TODO: Extract to separate test.
		//{
		//	name: "no config file exists",
		//	newEntry: ServerEntry{
		//		Name:    "test-server",
		//		Package: "modelcontextprotocol/test-server@latest",
		//	},
		//	shouldSetupConfig: false,
		//	isErrorExpected:   true,
		//	expectedErrMsg:    "config file cannot be found, run: 'mcpd init'",
		//},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			tempPath := filepath.Join(tempDir, flags.DefaultConfigFile)
			_, err := os.CreateTemp(tempDir, flags.DefaultConfigFile)
			if tc.shouldSetupConfig && tc.config != nil {
				createTestConfigFile(t, tempPath, *tc.config)
			}

			cfg, err := NewConfig(tempPath)
			require.NoError(t, err)
			err = cfg.AddServer(tc.newEntry)
			if tc.isErrorExpected {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrMsg)
				return
			}
			require.NoError(t, err)

			found := false
			for _, server := range cfg.Servers {
				if server.Name == tc.newEntry.Name && server.Package == tc.newEntry.Package {
					found = true
					assert.Equal(t, tc.newEntry.Tools, server.Tools)
					break
				}
			}
			assert.True(t, found, "Added server not found in config")
		})
	}
}

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name              string
		shouldSetupConfig bool
		configContent     *Config
		isErrorExpected   bool
		expectedErrMsg    string
	}{
		{
			name:              "load valid config",
			shouldSetupConfig: true,
			configContent: &Config{
				Servers: []ServerEntry{
					{Name: "test-server", Package: "modelcontextprotocol/test@v1.0.0"},
				},
			},
			isErrorExpected: false,
		},
		{
			name:              "config file does not exist",
			shouldSetupConfig: false,
			isErrorExpected:   true,
			expectedErrMsg:    "config file cannot be found, run: 'mcpd init'",
		},
		{
			name:              "load empty config",
			shouldSetupConfig: true,
			configContent: &Config{
				Servers: []ServerEntry{},
			},
			isErrorExpected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			tempPath := filepath.Join(tempDir, flags.DefaultConfigFile)

			if tc.shouldSetupConfig {
				createTestConfigFile(t, tempPath, *tc.configContent)
			} else {
				// Override global config flag to use test-specific file path that doesn't exist
				previousConfigFile := flags.ConfigFile
				flags.ConfigFile = "/foo/bar/baz.toml"
				defer func() { flags.ConfigFile = previousConfigFile }()
			}

			config, err := loadConfig(tempPath)

			if tc.isErrorExpected {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, len(tc.configContent.Servers), len(config.Servers))
			}
		})
	}
}

func TestConfig_SaveConfig(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "save config with servers",
			config: Config{
				Servers: []ServerEntry{
					{Name: "server1", Package: "modelcontextprotocol/server1@v1.0.0"},
					{Name: "server2", Package: "modelcontextprotocol/server2@latest", Tools: []string{"tool1", "tool2"}},
				},
			},
		},
		{
			name: "save empty config",
			config: Config{
				Servers: []ServerEntry{},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			tempPath := filepath.Join(tempDir, flags.DefaultConfigFile)
			tempFile, err := os.CreateTemp(tempDir, flags.DefaultConfigFile)
			require.NoError(t, err)
			tc.config.configFilePath = tempPath
			err = tc.config.saveConfig()
			require.NoError(t, err)

			assert.FileExists(t, tempFile.Name())
			loadedConfig, err := loadConfig(tempPath)
			require.NoError(t, err)
			assert.Equal(t, tc.config, loadedConfig)
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name            string
		config          Config
		isErrorExpected bool
		expectedErrMsg  string
	}{
		{
			name: "valid config",
			config: Config{
				Servers: []ServerEntry{
					{Name: "server1", Package: "modelcontextprotocol/server1@v1.0.0"},
					{Name: "server2", Package: "modelcontextprotocol/server2@latest"},
				},
			},
			isErrorExpected: false,
		},
		{
			name: "empty name",
			config: Config{
				Servers: []ServerEntry{
					{Name: "", Package: "modelcontextprotocol/server1@v1.0.0"},
				},
			},
			isErrorExpected: true,
			expectedErrMsg:  "server entry has empty name",
		},
		{
			name: "whitespace-only name",
			config: Config{
				Servers: []ServerEntry{
					{Name: "   ", Package: "modelcontextprotocol/server1@v1.0.0"},
				},
			},
			isErrorExpected: true,
			expectedErrMsg:  "server entry has empty name",
		},
		{
			name: "empty package",
			config: Config{
				Servers: []ServerEntry{
					{Name: "server1", Package: ""},
				},
			},
			isErrorExpected: true,
			expectedErrMsg:  "server entry has empty package",
		},
		{
			name: "duplicate servers",
			config: Config{
				Servers: []ServerEntry{
					{Name: "server1", Package: "modelcontextprotocol/server1@v1.0.0"},
					{Name: "server1", Package: "modelcontextprotocol/server1@v2.0.0"}, // Different version, same base
				},
			},
			isErrorExpected: true,
			expectedErrMsg:  "duplicate server entry",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.validate()

			if tc.isErrorExpected {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestStripVersion(t *testing.T) {
	tests := []struct {
		name     string
		pkg      string
		expected string
	}{
		{
			name:     "package with version",
			pkg:      "modelcontextprotocol/server@v1.0.0",
			expected: "modelcontextprotocol/server",
		},
		{
			name:     "package with latest",
			pkg:      "modelcontextprotocol/server@latest",
			expected: "modelcontextprotocol/server",
		},
		{
			name:     "package without version",
			pkg:      "modelcontextprotocol/server",
			expected: "modelcontextprotocol/server",
		},
		{
			name:     "package with multiple @ symbols",
			pkg:      "scope@org/package@v1.0.0",
			expected: "scope@org/package",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := stripVersion(tc.pkg)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestKeyFor(t *testing.T) {
	tests := []struct {
		name     string
		entry    ServerEntry
		expected serverKey
	}{
		{
			name: "basic server entry",
			entry: ServerEntry{
				Name:    "test-server",
				Package: "modelcontextprotocol/test-server@v1.0.0",
			},
			expected: serverKey{
				Name:    "test-server",
				Package: "modelcontextprotocol/test-server",
			},
		},
		{
			name: "server entry with tools",
			entry: ServerEntry{
				Name:    "tool-server",
				Package: "modelcontextprotocol/tool-server@latest",
				Tools:   []string{"tool1", "tool2"},
			},
			expected: serverKey{
				Name:    "tool-server",
				Package: "modelcontextprotocol/tool-server",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := keyFor(tc.entry)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestConfigFilePath_Default(t *testing.T) {
	t.Setenv(flags.EnvVarConfigFile, "")
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)

	// Reset global
	flags.ConfigFile = ""

	flags.InitFlags(fs)
	err := fs.Parse([]string{}) // No flags passed
	require.NoError(t, err)

	assert.Equal(t, flags.DefaultConfigFile, flags.ConfigFile)
}

func TestConfigFilePath_FromEnv(t *testing.T) {
	expected := "/custom/path/config.toml"
	t.Setenv(flags.EnvVarConfigFile, expected)

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)

	// Reset global
	flags.ConfigFile = ""

	flags.InitFlags(fs)
	err := fs.Parse([]string{}) // No flags passed
	require.NoError(t, err)

	assert.Equal(t, expected, flags.ConfigFile)
}

func TestConfigFilePath_FromFlag(t *testing.T) {
	expected := "/custom/path/flag.toml"
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)

	// Reset global
	flags.ConfigFile = ""

	flags.InitFlags(fs)
	err := fs.Parse([]string{"--" + flags.FlagNameConfigFile, expected})
	require.NoError(t, err)

	assert.Equal(t, expected, flags.ConfigFile)
}

// Helper functions for test setup and cleanup
func createTestConfigFile(t *testing.T, path string, config Config) {
	t.Helper()

	err := InitConfigFile(path)
	require.NoError(t, err)
	config.configFilePath = path
	err = config.saveConfig()
	require.NoError(t, err)
}
