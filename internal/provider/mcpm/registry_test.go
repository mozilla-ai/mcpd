package mcpm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/v2/internal/packages"
	"github.com/mozilla-ai/mcpd/v2/internal/registry/options"
	"github.com/mozilla-ai/mcpd/v2/internal/runtime"
)

// intPtr returns a pointer to an int
func intPtr(i int) *int {
	return &i
}

// loadTestDataServers loads MCPServers from a testdata JSON file for testing.
func loadTestDataServers(t *testing.T, filename string) MCPServers {
	t.Helper()

	testdataPath := filepath.Join("testdata", filename)
	require.FileExists(t, testdataPath, "testdata file should exist")

	data, err := os.ReadFile(testdataPath)
	require.NoError(t, err, "should be able to read testdata file")

	var servers MCPServers
	err = json.Unmarshal(data, &servers)
	require.NoError(t, err, "should be able to unmarshal testdata JSON")

	return servers
}

// loadTestDataJSON loads raw JSON from a testdata file for testing.
func loadTestDataJSON(t *testing.T, filename string) string {
	t.Helper()

	testdataPath := filepath.Join("testdata", filename)
	require.FileExists(t, testdataPath, "testdata file should exist")

	data, err := os.ReadFile(testdataPath)
	require.NoError(t, err, "should be able to read testdata file")

	return string(data)
}

// newTestLogger creates a test logger with debug level output.
func newTestLogger(t *testing.T) hclog.Logger {
	t.Helper()

	return hclog.New(&hclog.LoggerOptions{
		Level:  hclog.Debug,
		Output: os.Stderr,
		Name:   "test.mcpd",
	})
}

func TestRegistry_NewRegistry(t *testing.T) {
	mockJSON := loadTestDataJSON(t, "registry_mock.json")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(mockJSON))
		require.NoError(t, err)
	}))
	defer ts.Close()

	logger := newTestLogger(t)

	t.Run("successful creation", func(t *testing.T) {
		registry, err := NewRegistry(logger, ts.URL)
		require.NoError(t, err)
		require.NotNil(t, registry)
		require.Len(t, registry.mcpServers, 4, "Expected 4 servers in the map")
	})

	t.Run("http request failure", func(t *testing.T) {
		_, err := NewRegistry(logger, "http://nonexistent-domain.test/api/servers.json")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to fetch 'mcpm' registry data from URL")
	})

	t.Run("bad status code", func(t *testing.T) {
		badStatusServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer badStatusServer.Close()
		_, err := NewRegistry(logger, badStatusServer.URL+"/api/servers.json")
		require.Error(t, err)
		require.Contains(t, err.Error(), "received non-OK HTTP status from 'mcpm' registry for URL")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		invalidJSONServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte(`{"servers": "not an object"`))
			require.NoError(t, err)
		}))
		defer invalidJSONServer.Close()
		_, err := NewRegistry(logger, invalidJSONServer.URL+"/api/servers.json")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to unmarshal 'mcpm' registry JSON")
	})
}

func TestRegistry_Search(t *testing.T) {
	mockJSON := loadTestDataJSON(t, "registry_mock.json")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(mockJSON))
		require.NoError(t, err)
	}))
	defer ts.Close()

	logger := newTestLogger(t)
	registry, err := NewRegistry(logger, ts.URL)
	require.NoError(t, err)
	require.NotNil(t, registry)

	tests := []struct {
		name          string
		queryName     string
		filters       map[string]string
		expectedCount int
		expectedIDs   []string
		expectedEnv   map[string][]string // Map of ID to expected configurable env vars
		expectedArgs  map[string][]string // Map of ID to expected configurable cmd line args
	}{
		{
			name:          "basic search for time",
			queryName:     "time",
			filters:       nil,
			expectedCount: 1,
			expectedIDs:   []string{"time"},
			expectedEnv:   map[string][]string{"time": {}},
			expectedArgs:  map[string][]string{"time": {"--local-timezone"}},
		},
		{
			name:          "case insensitive search",
			queryName:     "TIME",
			filters:       nil,
			expectedCount: 1,
			expectedIDs:   []string{"time"},
			expectedEnv:   map[string][]string{"time": {}},
			expectedArgs:  map[string][]string{"time": {"--local-timezone"}},
		},
		{
			name:          "unsupported runtime filtered out",
			queryName:     "GitHub",
			filters:       nil,
			expectedCount: 0,
			expectedIDs:   nil,
			expectedEnv:   nil,
		},
		{
			name:          "search by display name",
			queryName:     "math server",
			filters:       nil,
			expectedCount: 1,
			expectedIDs:   []string{"math"},
			expectedEnv:   map[string][]string{"math": {}},
			expectedArgs:  map[string][]string{"math": {}},
		},
		{
			name:          "runtime filter uvx",
			queryName:     "*",
			filters:       map[string]string{"runtime": "uvx"},
			expectedCount: 2,
			expectedIDs:   []string{"time", "math"},
			expectedEnv:   map[string][]string{"time": {}, "math": {}},
			expectedArgs:  map[string][]string{"time": {"--local-timezone"}, "math": {}},
		},
		{
			name:          "tool filter add",
			queryName:     "*",
			filters:       map[string]string{"tools": "add"},
			expectedCount: 1,
			expectedIDs:   []string{"math"},
			expectedEnv:   map[string][]string{"math": {}},
			expectedArgs:  map[string][]string{"math": {}},
		},
		{
			name:          "nonexistent query",
			queryName:     "nonexistent",
			filters:       nil,
			expectedCount: 0,
			expectedIDs:   nil,
			expectedEnv:   nil,
		},
		{
			name:          "combined filters",
			queryName:     "*",
			filters:       map[string]string{"runtime": "uvx", "tools": "convert_time"},
			expectedCount: 1,
			expectedIDs:   []string{"time"},
			expectedEnv:   map[string][]string{"time": {}},
			expectedArgs:  map[string][]string{"time": {"--local-timezone"}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			results, err := registry.Search(tc.queryName, tc.filters)
			require.NoError(t, err)
			require.Len(t, results, tc.expectedCount, "Unexpected result count")

			if tc.expectedCount > 0 {
				resultIDs := make([]string, len(results))
				for i, result := range results {
					resultIDs[i] = result.ID
				}
				require.ElementsMatch(t, tc.expectedIDs, resultIDs, "Unexpected result IDs")

				// Verify argument extraction for each result
				for _, result := range results {
					if expectedEnvVars, ok := tc.expectedEnv[result.ID]; ok {
						envVars := extractTestEnvVarNames(t, result.Arguments)
						require.ElementsMatch(t, expectedEnvVars, envVars,
							"Unexpected env vars for %s", result.ID)
					}
					if expectedArgs, ok := tc.expectedArgs[result.ID]; ok {
						cliArgs := extractTestCLIArgNames(t, result.Arguments)
						require.ElementsMatch(t, expectedArgs, cliArgs,
							"Unexpected CLI args for %s", result.ID)
					}
				}
			}
		})
	}
}

func TestRegistry_Resolve(t *testing.T) {
	mockJSON := loadTestDataJSON(t, "registry_mock.json")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(mockJSON))
		require.NoError(t, err)
	}))
	defer ts.Close()

	logger := newTestLogger(t)
	registry, err := NewRegistry(logger, ts.URL)
	require.NoError(t, err)
	require.NotNil(t, registry)

	tests := []struct {
		name         string
		id           string
		version      string
		expectError  bool
		expectedID   string
		expectedEnv  []string // Expected configurable env vars for the single result
		expectedArgs []string
	}{
		{
			name:         "existing package with empty version",
			id:           "time",
			version:      "",
			expectError:  false,
			expectedID:   "time",
			expectedEnv:  []string{},
			expectedArgs: []string{"--local-timezone"},
		},
		{
			name:         "existing package with latest version",
			id:           "time",
			version:      "latest",
			expectError:  false,
			expectedID:   "time",
			expectedEnv:  []string{},
			expectedArgs: []string{"--local-timezone"},
		},
		{
			name:        "nonexistent package",
			id:          "nonexistent-package",
			version:     "",
			expectError: true,
			expectedID:  "",
			expectedEnv: nil,
		},
		{
			name:         "version ignored with warning",
			id:           "math",
			version:      "1.0.0",
			expectError:  false,
			expectedID:   "math",
			expectedEnv:  []string{},
			expectedArgs: []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := registry.Resolve(tc.id, options.WithResolveVersion(tc.version))

			if tc.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expectedID, result.ID)

			envVars := extractTestEnvVarNames(t, result.Arguments)
			require.ElementsMatch(t, tc.expectedEnv, envVars, "Unexpected env vars")

			cliArgs := extractTestCLIArgNames(t, result.Arguments)
			require.ElementsMatch(t, tc.expectedArgs, cliArgs, "Unexpected CLI args")
		})
	}
}

func TestRegistry_ExtractArgumentMetadata_RealRegistry(t *testing.T) {
	t.Parallel()

	// Test with real MCPM registry data to ensure we handle real-world scenarios
	servers := loadTestDataServers(t, "registry_real.json")
	require.Greater(t, len(servers), 0, "Real registry should have servers")

	// Test a few known servers from the real registry
	testCases := []struct {
		name         string
		serverName   string
		description  string
		expectedArgs []string
	}{
		{
			name:         "memory server",
			serverName:   "memory",
			description:  "Real memory server should not cause extraction errors",
			expectedArgs: []string{},
		},
		{
			name:         "filesystem server",
			serverName:   "filesystem",
			description:  "Real filesystem server should extract environment variables correctly",
			expectedArgs: []string{},
		},
		{
			name:         "neo4j memory server",
			serverName:   "mcp-neo4j-memory",
			description:  "Neo4j server should only extract flags that are actually used",
			expectedArgs: []string{"--db-url", "--username", "--password"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			t.Logf("Test description: %s", tc.description)

			server, exists := servers[tc.serverName]
			if !exists {
				t.Skipf("Server %q not found in real registry data", tc.serverName)
				return
			}

			// Extract arguments - should not panic or error
			result := extractArgumentMetadata(server, runtime.DefaultSupportedRuntimes())

			// Basic validation - result should be a valid map
			require.NotNil(t, result, "Result should not be nil")

			// Log what we found for debugging
			t.Logf("Server %q extracted %d arguments", tc.serverName, len(result))
			for argName, metadata := range result {
				t.Logf("  %s: %s (%s)", argName, metadata.VariableType, metadata.Description)
			}

			// Check expected CLI args if specified
			if len(tc.expectedArgs) > 0 {
				cliArgs := extractTestCLIArgNames(t, result)
				require.ElementsMatch(t, tc.expectedArgs, cliArgs,
					"Server %q: Expected CLI args %v but got %v", tc.serverName, tc.expectedArgs, cliArgs)
			}
		})
	}
}

func TestRegistry_ExtractArgumentMetadata_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		server   MCPServer
		expected map[string]packages.ArgumentMetadata
	}{
		{
			name: "uvx and docker installations with placeholders",
			server: MCPServer{
				Name: "test-server",
				Arguments: Arguments{
					"ENV_VAR": {Description: "Test env var", Required: true, Example: "test"},
				},
				Installations: map[string]Installation{
					"uvx": {
						Type:    "uvx",
						Command: "uvx",
						Args:    []string{"package", "--config=${ENV_VAR}"},
					},
					"docker": {
						Type:    "docker",
						Command: "docker",
						Args:    []string{"run", "--rm", "-v", "/tmp:/tmp", "image"},
					},
				},
			},
			expected: map[string]packages.ArgumentMetadata{
				"--config": {
					Name:         "--config",
					VariableType: packages.VariableTypeArg,
					Required:     true,
					Description:  "Test env var",
					Example:      "test",
				},
			},
		},
		{
			name: "env var placeholder discovered",
			server: MCPServer{
				Name: "test-server",
				Arguments: Arguments{
					"API_KEY": {Description: "API key", Required: true, Example: "key123"},
				},
				Installations: map[string]Installation{
					"npx": {
						Type:    "npx",
						Command: "npx",
						Args:    []string{"server", "${API_KEY}"},
					},
				},
			},
			expected: map[string]packages.ArgumentMetadata{
				"API_KEY": {
					Name:         "API_KEY",
					VariableType: packages.VariableTypePositionalArg,
					Required:     true,
					Description:  "API key",
					Example:      "key123",
					Position:     intPtr(1),
				},
			},
		},
		{
			name: "env var placeholder looking ahead",
			server: MCPServer{
				Name: "test-server",
				Arguments: Arguments{
					"CONFIG_PATH": {Description: "Config file path", Required: true, Example: "/config.json"},
				},
				Installations: map[string]Installation{
					"npx": {
						Type:    "npx",
						Command: "npx",
						Args:    []string{"server", "--config", "${CONFIG_PATH}"},
					},
				},
			},
			expected: map[string]packages.ArgumentMetadata{
				"--config": {
					Name:         "--config",
					VariableType: packages.VariableTypeArg,
					Required:     true,
					Description:  "Config file path",
					Example:      "/config.json",
				},
			},
		},
		{
			name: "unsupported runtime skipped",
			server: MCPServer{
				Name: "test-server",
				Arguments: Arguments{
					"VAR": {Description: "Test var", Required: true},
				},
				Installations: map[string]Installation{
					"unsupported": {
						Type:    "unsupported",
						Command: "unsupported",
						Args:    []string{"--flag=${VAR}"},
					},
				},
			},
			expected: map[string]packages.ArgumentMetadata{},
		},
		{
			name: "cli argument types",
			server: MCPServer{
				Name: "test-server",
				Arguments: Arguments{
					"DEBUG": {Description: "Debug mode", Required: false},
				},
				Installations: map[string]Installation{
					"npx": {
						Type:    "npx",
						Command: "npx",
						Args:    []string{"server", "--debug"},
					},
				},
			},
			expected: map[string]packages.ArgumentMetadata{
				"--debug": {
					Name:         "--debug",
					VariableType: packages.VariableTypeArgBool,
					Required:     false,
					Description:  "",
				},
			},
		},
		{
			name: "env vars declared in installation env",
			server: MCPServer{
				Name: "test-server",
				Arguments: Arguments{
					"API_TOKEN": {Description: "API token", Required: true, Example: "token123"},
				},
				Installations: map[string]Installation{
					"npx": {
						Type:    "npx",
						Command: "npx",
						Args:    []string{"server"},
						Env: map[string]string{
							"API_TOKEN": "${API_TOKEN}",
						},
					},
				},
			},
			expected: map[string]packages.ArgumentMetadata{
				"API_TOKEN": {
					Name:         "API_TOKEN",
					VariableType: packages.VariableTypeEnv,
					Required:     true,
					Description:  "API token",
					Example:      "token123",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := extractArgumentMetadata(tc.server, runtime.DefaultSupportedRuntimes())
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestRegistry_ExtractArgumentMetadata_WithTestdata(t *testing.T) {
	testCases := []struct {
		name         string
		testdataFile string
		packageName  string
		description  string
		expected     map[string]packages.ArgumentMetadata
	}{
		{
			name:         "filesystem server with positional placeholders",
			testdataFile: "arg_classify_filesystem.json",
			packageName:  "@modelcontextprotocol/server-filesystem",
			description:  "Positional placeholders should be extracted as positional arguments",
			expected: map[string]packages.ArgumentMetadata{
				"USER_FILESYSTEM_DIRECTORY": {
					Name:         "USER_FILESYSTEM_DIRECTORY",
					VariableType: packages.VariableTypePositionalArg,
					Required:     true,
					Description:  "The base directory that the server will have access to",
					Example:      "/Users/username/Documents",
					Position:     intPtr(1),
				},
				"USER_FILESYSTEM_ALLOWED_DIR": {
					Name:         "USER_FILESYSTEM_ALLOWED_DIR",
					VariableType: packages.VariableTypePositionalArg,
					Required:     false,
					Description:  "Additional allowed directory for file access",
					Example:      "/Users/username/Projects",
					Position:     intPtr(2),
				},
			},
		},
		{
			name:         "flag with embedded placeholder",
			testdataFile: "arg_classify_flags.json",
			packageName:  "flag-with-placeholder",
			description:  "Flag containing placeholder should extract the flag as CLI argument",
			expected: map[string]packages.ArgumentMetadata{
				"--timezone": {
					Name:         "--timezone",
					VariableType: packages.VariableTypeArg,
					Required:     false,
					Description:  "Timezone setting for the server",
					Example:      "America/New_York",
				},
			},
		},
		{
			name:         "flag looking ahead to next argument",
			testdataFile: "arg_classify_flags.json",
			packageName:  "flag-looking-ahead",
			description:  "Flag followed by placeholder in next arg should extract only the flag",
			expected: map[string]packages.ArgumentMetadata{
				"--config": {
					Name:         "--config",
					VariableType: packages.VariableTypeArg,
					Required:     true,
					Description:  "Path to configuration file",
					Example:      "/path/to/config.json",
				},
			},
		},
		{
			name:         "positional placeholder only",
			testdataFile: "arg_classify_env.json",
			packageName:  "positional-placeholder",
			description:  "Positional placeholder should be classified as positional argument",
			expected: map[string]packages.ArgumentMetadata{
				"DATA_DIR": {
					Name:         "DATA_DIR",
					VariableType: packages.VariableTypePositionalArg,
					Required:     true,
					Description:  "Directory for data storage",
					Example:      "/path/to/data",
					Position:     intPtr(1),
				},
			},
		},
		{
			name:         "environment variable only",
			testdataFile: "arg_classify_env.json",
			packageName:  "env-only",
			description:  "Variable used only in env section should remain as environment variable",
			expected: map[string]packages.ArgumentMetadata{
				"API_KEY": {
					Name:         "API_KEY",
					VariableType: packages.VariableTypeEnv,
					Required:     true,
					Description:  "API key for authentication",
					Example:      "sk-1234567890",
				},
			},
		},
		{
			name:         "mixed environment and flag usage",
			testdataFile: "arg_classify_mixed.json",
			packageName:  "mixed-env-and-flag",
			description:  "Variable used in both env and flag should extract both, with env taking precedence",
			expected: map[string]packages.ArgumentMetadata{
				"--db-url": {
					Name:         "--db-url",
					VariableType: packages.VariableTypeArg,
					Required:     true,
					Description:  "Database connection URL",
					Example:      "postgresql://user:pass@localhost:5432/db",
				},
				"DATABASE_URL": {
					Name:         "DATABASE_URL",
					VariableType: packages.VariableTypeEnv,
					Required:     true,
					Description:  "Database connection URL",
					Example:      "postgresql://user:pass@localhost:5432/db",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Test description: %s", tc.description)

			// Load testdata servers
			servers := loadTestDataServers(t, tc.testdataFile)

			// Get the specific server we're testing
			server, exists := servers[tc.packageName]
			require.True(t, exists, "Package %q not found in testdata file %q", tc.packageName, tc.testdataFile)

			// Extract arguments using the fixed logic
			result := extractArgumentMetadata(server, runtime.DefaultSupportedRuntimes())

			// Verify expected arguments are present and correctly classified
			require.Equal(t, len(tc.expected), len(result),
				"Unexpected number of arguments extracted for %s", tc.packageName)

			for expectedKey, expectedMeta := range tc.expected {
				actualMeta, found := result[expectedKey]
				require.True(t, found, "Expected argument %q not found in results", expectedKey)
				require.Equal(t, expectedMeta, actualMeta,
					"Argument metadata mismatch for %q", expectedKey)
			}
		})
	}
}

func TestRegistry_ExtractArgumentMetadata_SyntheticCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		description string
		server      MCPServer
		expected    map[string]packages.ArgumentMetadata
	}{
		{
			name:        "flag with embedded placeholder",
			description: "Flag containing placeholder should extract the flag as CLI argument",
			server: MCPServer{
				Name: "timezone-server",
				Arguments: Arguments{
					"TZ": {Description: "Timezone setting", Required: false, Example: "America/New_York"},
				},
				Installations: map[string]Installation{
					"npx": {Type: "npx", Command: "npx", Args: []string{"timezone-server", "--timezone=${TZ}"}},
				},
			},
			expected: map[string]packages.ArgumentMetadata{
				"--timezone": {
					Name:         "--timezone",
					VariableType: packages.VariableTypeArg,
					Required:     false,
					Description:  "Timezone setting",
					Example:      "America/New_York",
				},
			},
		},
		{
			name:        "positional placeholder",
			description: "Positional placeholder should be classified as positional argument",
			server: MCPServer{
				Name: "filesystem-server",
				Arguments: Arguments{
					"BASE_DIR": {Description: "Base directory", Required: true, Example: "/path/to/files"},
				},
				Installations: map[string]Installation{
					"npx": {Type: "npx", Command: "npx", Args: []string{"filesystem-server", "${BASE_DIR}"}},
				},
			},
			expected: map[string]packages.ArgumentMetadata{
				"BASE_DIR": {
					Name:         "BASE_DIR",
					VariableType: packages.VariableTypePositionalArg,
					Required:     true,
					Description:  "Base directory",
					Example:      "/path/to/files",
					Position:     intPtr(1),
				},
			},
		},
		{
			name:        "environment variable only",
			description: "Variable used only in env section should remain as environment variable",
			server: MCPServer{
				Name: "api-server",
				Arguments: Arguments{
					"API_SECRET": {Description: "API secret", Required: true, Example: "sk-abcd1234"},
				},
				Installations: map[string]Installation{
					"npx": {
						Type: "npx", Command: "npx", Args: []string{"api-server"},
						Env: map[string]string{"API_SECRET": "${API_SECRET}"},
					},
				},
			},
			expected: map[string]packages.ArgumentMetadata{
				"API_SECRET": {
					Name:         "API_SECRET",
					VariableType: packages.VariableTypeEnv,
					Required:     true,
					Description:  "API secret",
					Example:      "sk-abcd1234",
				},
			},
		},
		{
			name:        "mixed env and flag usage",
			description: "Variable used in both env and flag should extract both",
			server: MCPServer{
				Name: "database-server",
				Arguments: Arguments{
					"DB_URL": {Description: "Database URL", Required: true, Example: "postgresql://localhost:5432/db"},
				},
				Installations: map[string]Installation{
					"npx": {
						Type: "npx", Command: "npx", Args: []string{"database-server", "--db=${DB_URL}"},
						Env: map[string]string{"DB_URL": "${DB_URL}"},
					},
				},
			},
			expected: map[string]packages.ArgumentMetadata{
				"DB_URL": {
					Name:         "DB_URL",
					VariableType: packages.VariableTypeEnv,
					Required:     true,
					Description:  "Database URL",
					Example:      "postgresql://localhost:5432/db",
				},
				"--db": {
					Name:         "--db",
					VariableType: packages.VariableTypeArg,
					Required:     true,
					Description:  "Database URL",
					Example:      "postgresql://localhost:5432/db",
				},
			},
		},
		{
			name:        "flag looking ahead",
			description: "Flag followed by placeholder should extract only the flag",
			server: MCPServer{
				Name: "config-server",
				Arguments: Arguments{
					"CONFIG_FILE": {Description: "Config file path", Required: true, Example: "/etc/config.json"},
				},
				Installations: map[string]Installation{
					"npx": {Type: "npx", Command: "npx", Args: []string{"config-server", "--config", "${CONFIG_FILE}"}},
				},
			},
			expected: map[string]packages.ArgumentMetadata{
				"--config": {
					Name:         "--config",
					VariableType: packages.VariableTypeArg,
					Required:     true,
					Description:  "Config file path",
					Example:      "/etc/config.json",
				},
			},
		},
		{
			name:        "boolean flag without placeholder",
			description: "Boolean flags should be classified as boolean CLI arguments",
			server: MCPServer{
				Name: "debug-server",
				Arguments: Arguments{
					"VERBOSE": {Description: "Verbose output", Required: false},
				},
				Installations: map[string]Installation{
					"npx": {Type: "npx", Command: "npx", Args: []string{"debug-server", "--verbose"}},
				},
			},
			expected: map[string]packages.ArgumentMetadata{
				"--verbose": {
					Name:         "--verbose",
					VariableType: packages.VariableTypeArgBool,
					Required:     false,
					Description:  "",
				},
			},
		},
		{
			name:        "ignored flags not extracted",
			description: "Flags that should be ignored should not appear in extracted arguments",
			server: MCPServer{
				Name: "npx-server",
				Arguments: Arguments{
					"CONFIG": {Description: "Config option", Required: true},
				},
				Installations: map[string]Installation{
					"npx": {Type: "npx", Command: "npx", Args: []string{"-y", "package", "--config=${CONFIG}"}},
				},
			},
			expected: map[string]packages.ArgumentMetadata{
				"--config": {
					Name:         "--config",
					VariableType: packages.VariableTypeArg,
					Required:     true,
					Description:  "Config option",
					Example:      "",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			t.Logf("Test description: %s", tc.description)

			result := extractArgumentMetadata(tc.server, runtime.DefaultSupportedRuntimes())
			require.Equal(t, tc.expected, result, "Argument classification mismatch")
		})
	}
}

func TestRegistry_ShouldIgnoreFlag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		runtime  runtime.Runtime
		flag     string
		expected bool
	}{
		{runtime.Docker, "--rm", true},
		{runtime.Docker, "--name", true},
		{runtime.Docker, "--volume", true},
		{runtime.Docker, "-v", true},
		{runtime.Docker, "--network", true},
		{runtime.Docker, "--detach", true},
		{runtime.Docker, "-d", true},
		{runtime.Docker, "-i", true},
		{runtime.Docker, "--other", false},
		{runtime.Python, "-m", true},
		{runtime.Python, "--debug", false},
		{runtime.NPX, "-y", true},
		{runtime.NPX, "--other", false},
		{runtime.UVX, "--experimental", false},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%s/%s", tc.runtime, tc.flag), func(t *testing.T) {
			t.Parallel()
			result := shouldIgnoreFlag(tc.runtime, tc.flag)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestRegistry_Tool_ToDomainType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   Tool
		want    packages.Tool
		wantErr string
	}{
		{
			name: "valid tool",
			input: Tool{
				Name:        "t1",
				Title:       "Tool One",
				Description: "Test tool",
				InputSchema: JSONSchema{
					Type: "object",
					Properties: map[string]any{
						"foo": map[string]any{"type": "string"},
					},
					Required: []string{"foo"},
				},
			},
			want: packages.Tool{
				Name:        "t1",
				Title:       "Tool One",
				Description: "Test tool",
				InputSchema: packages.JSONSchema{
					Type: "object",
					Properties: map[string]any{
						"foo": map[string]any{"type": "string"},
					},
					Required: []string{"foo"},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual, err := tc.input.ToDomainType()

			switch {
			case tc.wantErr != "":
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
			default:
				require.NoError(t, err)
				require.Equal(t, tc.want, actual)
			}
		})
	}
}

func TestRegistry_Tools_ToDomainType(t *testing.T) {
	t.Parallel()

	validTool := Tool{
		Name:        "valid",
		Title:       "Valid",
		Description: "ok",
		InputSchema: JSONSchema{
			Type:       "object",
			Properties: map[string]any{},
			Required:   []string{},
		},
	}

	t.Run("all valid", func(t *testing.T) {
		t.Parallel()

		in := Tools{validTool, validTool}
		out, err := in.ToDomainType()
		require.NoError(t, err)
		require.Len(t, out, 2)
		require.Equal(t, "valid", out[0].Name)
	})
}

func TestRegistry_ExtractArgumentMetadata_ComprehensiveScenarios(t *testing.T) {
	t.Parallel()

	server := MCPServer{
		Name: "comprehensive-server",
		Arguments: Arguments{
			"DB_URL": {
				Description: "Database connection URL",
				Required:    true,
				Example:     "postgresql://localhost:5432/db",
			},
			"API_SECRET":   {Description: "API secret for authentication", Required: true, Example: "sk-secret123"},
			"BASE_DIR":     {Description: "Base directory for files", Required: true, Example: "/path/to/files"},
			"OPTIONAL_DIR": {Description: "Optional directory", Required: false, Example: "/path/to/optional"},
			"DEBUG_MODE":   {Description: "Enable debug output", Required: false},
			"LOG_LEVEL":    {Description: "Log level setting", Required: false, Example: "info"},
		},
		Installations: map[string]Installation{
			"npx": {
				Type:    "npx",
				Command: "npx",
				Args: []string{
					"comprehensive-server",
					"${BASE_DIR}",        // positional arg 1
					"${OPTIONAL_DIR}",    // positional arg 2
					"--db-url=${DB_URL}", // flag with embedded placeholder
					"--log-level",        // flag looking ahead
					"${LOG_LEVEL}",       // value for previous flag (positional arg 3)
					"--debug",            // boolean flag
				},
				Env: map[string]string{
					"API_SECRET": "${API_SECRET}", // environment variable
					"DB_URL":     "${DB_URL}",     // same placeholder used in both env and args
				},
			},
		},
	}

	expected := map[string]packages.ArgumentMetadata{
		// Environment variables
		"API_SECRET": {
			Name:         "API_SECRET",
			VariableType: packages.VariableTypeEnv,
			Required:     true,
			Description:  "API secret for authentication",
			Example:      "sk-secret123",
		},
		"DB_URL": {
			Name:         "DB_URL",
			VariableType: packages.VariableTypeEnv, // env takes precedence over flag
			Required:     true,
			Description:  "Database connection URL",
			Example:      "postgresql://localhost:5432/db",
		},
		// Command-line flags
		"--db-url": {
			Name:         "--db-url",
			VariableType: packages.VariableTypeArg,
			Required:     true,
			Description:  "Database connection URL",
			Example:      "postgresql://localhost:5432/db",
		},
		"--log-level": {
			Name:         "--log-level",
			VariableType: packages.VariableTypeArg,
			Required:     false,
			Description:  "Log level setting",
			Example:      "info",
		},
		"--debug": {
			Name:         "--debug",
			VariableType: packages.VariableTypeArgBool,
			Required:     false,
			Description:  "",
		},
		// Positional arguments
		"BASE_DIR": {
			Name:         "BASE_DIR",
			VariableType: packages.VariableTypePositionalArg,
			Required:     true,
			Description:  "Base directory for files",
			Example:      "/path/to/files",
			Position:     intPtr(1),
		},
		"OPTIONAL_DIR": {
			Name:         "OPTIONAL_DIR",
			VariableType: packages.VariableTypePositionalArg,
			Required:     false,
			Description:  "Optional directory",
			Example:      "/path/to/optional",
			Position:     intPtr(2),
		},
	}

	t.Logf("Testing comprehensive scenario with env vars, flags, and positional arguments")
	result := extractArgumentMetadata(server, runtime.DefaultSupportedRuntimes())

	require.Equal(t, len(expected), len(result),
		"Unexpected number of arguments extracted")

	for expectedKey, expectedMeta := range expected {
		actualMeta, found := result[expectedKey]
		require.True(t, found, "Expected argument %q not found in results", expectedKey)
		require.Equal(t, expectedMeta, actualMeta,
			"Argument metadata mismatch for %q", expectedKey)
	}

	// Verify positional arguments are correctly ordered using the compositional approach
	positionalArgs := packages.Arguments(result).FilterBy(packages.PositionalArgument)
	orderedPositional := positionalArgs.Ordered()

	require.Len(t, orderedPositional, 2, "Expected 2 positional arguments")
	require.Equal(t, "BASE_DIR", orderedPositional[0].Name)
	require.Equal(t, 1, *orderedPositional[0].Position)
	require.Equal(t, "OPTIONAL_DIR", orderedPositional[1].Name)
	require.Equal(t, 2, *orderedPositional[1].Position)

	// Verify the new Ordered() method: positional args first, then others alphabetically
	allOrdered := packages.Arguments(result).Ordered()
	require.Len(t, allOrdered, 7, "Expected all 7 arguments in ordered list")

	// First two should be positional arguments in position order
	require.Equal(t, "BASE_DIR", allOrdered[0].Name)
	require.Equal(t, packages.VariableTypePositionalArg, allOrdered[0].VariableType)
	require.Equal(t, "OPTIONAL_DIR", allOrdered[1].Name)
	require.Equal(t, packages.VariableTypePositionalArg, allOrdered[1].VariableType)

	// Rest should be non-positional in alphabetical order by name
	expectedOrder := []string{"--db-url", "--debug", "--log-level", "API_SECRET", "DB_URL"}
	for i, expectedName := range expectedOrder {
		require.Equal(t, expectedName, allOrdered[i+2].Name,
			"Expected %s at position %d but got %s", expectedName, i+2, allOrdered[i+2].Name)
	}
}

// extractTestEnvVarNames extracts environment variable names from Arguments for testing.
func extractTestEnvVarNames(t *testing.T, args packages.Arguments) []string {
	t.Helper()
	var envVars []string
	for key, arg := range args {
		if arg.VariableType == packages.VariableTypeEnv {
			envVars = append(envVars, key)
		}
	}
	return envVars
}

// extractTestCLIArgNames extracts CLI argument names from Arguments for testing.
func extractTestCLIArgNames(t *testing.T, args packages.Arguments) []string {
	t.Helper()
	var cliArgs []string
	for key, arg := range args {
		if arg.VariableType == packages.VariableTypeArg || arg.VariableType == packages.VariableTypeArgBool ||
			arg.VariableType == packages.VariableTypePositionalArg {
			cliArgs = append(cliArgs, key)
		}
	}
	return cliArgs
}
