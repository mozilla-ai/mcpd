package mozilla_ai

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/v2/internal/packages"
	"github.com/mozilla-ai/mcpd/v2/internal/registry/options"
	"github.com/mozilla-ai/mcpd/v2/internal/runtime"
)

// loadTestDataRegistry loads MCPRegistry from a testdata JSON file for testing.
func loadTestDataRegistry(t *testing.T, filename string) MCPRegistry {
	t.Helper()

	testdataPath := filepath.Join("testdata", filename)
	require.FileExists(t, testdataPath, "testdata file should exist")

	data, err := os.ReadFile(testdataPath)
	require.NoError(t, err, "should be able to read testdata file")

	var registry MCPRegistry
	err = json.Unmarshal(data, &registry)
	require.NoError(t, err, "should be able to unmarshal testdata JSON")

	return registry
}

// newTestLogger creates a test logger with debug level output.
func newTestLogger(t *testing.T) hclog.Logger {
	t.Helper()

	return hclog.New(&hclog.LoggerOptions{
		Level:  hclog.Debug,
		Output: os.Stderr,
		Name:   "test.mozilla-ai",
	})
}

func TestRegistry_ID(t *testing.T) {
	t.Parallel()

	logger := newTestLogger(t)
	registry, err := NewRegistry(logger, "", runtime.WithSupportedRuntimes(runtime.NPX, runtime.UVX))
	require.NoError(t, err)
	require.NotNil(t, registry)
	require.Equal(t, RegistryName, registry.ID())
	require.Equal(t, "mozilla-ai", registry.ID())
}

func TestRegistry_NewRegistry_EmbeddedData(t *testing.T) {
	t.Parallel()

	// No URL so we use embedded JSON data.
	registry, err := NewRegistry(
		hclog.NewNullLogger(),
		"",
		runtime.WithSupportedRuntimes(runtime.NPX, runtime.UVX),
	)
	require.NoError(t, err)
	require.NotNil(t, registry)
	require.Greater(t, len(registry.mcpServers), 0)
	require.Contains(t, registry.mcpServers, "filesystem")
	require.Contains(t, registry.mcpServers, "memory")
	require.Contains(t, registry.mcpServers, "mcp-discord")
}

func TestRegistry_NewRegistry_NoSupportedRuntimes(t *testing.T) {
	t.Parallel()

	_, err := NewRegistry(
		hclog.NewNullLogger(),
		"",
		runtime.WithSupportedRuntimes(runtime.Docker),
	)
	require.Error(t, err)
	require.EqualError(t, err, "no supported runtimes for mozilla-ai registry: requires at least one of: npx, uvx")
}

func TestRegistry_Resolve_EmbeddedServer(t *testing.T) {
	t.Parallel()

	registry, err := NewRegistry(
		hclog.NewNullLogger(),
		"",
		runtime.WithSupportedRuntimes(runtime.UVX, runtime.NPX),
	)
	require.NoError(t, err)
	require.Contains(t, registry.mcpServers, "filesystem")
	pkg, transformed := registry.serverForID("filesystem")
	require.True(t, transformed)
	require.NotNil(t, pkg)
	require.Equal(t, "filesystem", pkg.Name)
	require.Equal(t, RegistryName, pkg.Source)
	require.NotEmpty(t, pkg.Description)
	require.NotEmpty(t, pkg.License)
	require.Greater(t, len(pkg.Tools), 0, "Should have tools")
	require.Greater(t, len(pkg.Installations), 0, "Should have installation details")
}

func TestRegistry_Resolve_NonExistentPackage(t *testing.T) {
	t.Parallel()

	registry, err := NewRegistry(
		hclog.NewNullLogger(),
		"",
		runtime.WithSupportedRuntimes(runtime.UVX, runtime.NPX),
	)
	require.NoError(t, err)
	_, err = registry.Resolve("nonexistent-server-that-does-not-exist")
	require.Error(t, err)
	require.EqualError(t, err, "failed to build package result for 'nonexistent-server-that-does-not-exist'")
}

func TestRegistry_Search(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		id      string
		filters map[string]string
		opts    []options.SearchOption
	}{
		{
			name:    "id only",
			id:      "filesystem",
			filters: nil,
			opts:    nil,
		},
		{
			name: "single filter",
			id:   "filesystem",
			filters: map[string]string{
				"tags": "filesystem",
			},
			opts: nil,
		},
		{
			name: "single filter, multiple values",
			id:   "filesystem",
			filters: map[string]string{
				"tags": "filesystem,file operations",
			},
			opts: nil,
		},
		{
			name: "multiple filter",
			id:   "filesystem",
			filters: map[string]string{
				"tags":       "filesystem,file operations",
				"categories": "System Tools",
			},
			opts: nil,
		},
		{
			name: "source exact match",
			id:   "filesystem",
			filters: map[string]string{
				"tags": "filesystem,file operations",
			},
			opts: []options.SearchOption{
				options.WithSearchSource(RegistryName),
			},
		},
		{
			name: "source partial match",
			id:   "filesystem",
			filters: map[string]string{
				"tags": "filesystem,file operations",
			},
			opts: []options.SearchOption{
				options.WithSearchSource("mozilla"),
			},
		},
	}

	ensureFound := func(t *testing.T, results []packages.Server, name string) {
		t.Helper()

		found := false
		for _, result := range results {
			if result.Name == name {
				found = true
				break
			}
		}
		require.True(t, found)
	}

	registry, err := NewRegistry(
		hclog.NewNullLogger(),
		"",
		runtime.WithSupportedRuntimes(runtime.UVX, runtime.NPX),
	)
	require.NoError(t, err)

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			results, err := registry.Search(tc.id, tc.filters, tc.opts...)
			require.NoError(t, err)
			ensureFound(t, results, tc.id)
		})
	}

	results, err := registry.Search("filesystem", nil)
	require.NoError(t, err)
	ensureFound(t, results, "filesystem")

	results, err = registry.Search("filesystem", map[string]string{"tags": "filesystem"})
	require.NoError(t, err)
	ensureFound(t, results, "filesystem")

	results, err = registry.Search("filesystem", map[string]string{"tags": "filesystem,file operations"})
	require.NoError(t, err)
	ensureFound(t, results, "filesystem")

	results, err = registry.Search("filesystem", map[string]string{"tags": "filesystem,file operations"})
	require.NoError(t, err)
	ensureFound(t, results, "filesystem")
}

func TestRegistry_Tools_ToDomainType(t *testing.T) {
	t.Parallel()

	tools := Tools{
		{
			Name:        "test_tool",
			Description: "A test tool",
			Title:       "Test Tool",
		},
	}

	result, err := tools.ToDomainType()
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, "test_tool", result[0].Name)
	require.Equal(t, "A test tool", result[0].Description)
	require.Equal(t, "Test Tool", result[0].Title)
}

func TestRegistry_Arguments_ToDomainType(t *testing.T) {
	t.Parallel()

	args := Arguments{
		"TEST_ENV": {
			Name:        "TEST_ENV",
			Description: "Test environment variable",
			Required:    true,
			Type:        ArgumentEnv,
		},
		"TEST_ARG": {
			Name:        "TEST_ARG",
			Description: "Test command argument",
			Required:    false,
			Type:        ArgumentValue,
		},
	}

	result, err := args.ToDomainType()
	require.NoError(t, err)
	require.Len(t, result, 2)

	require.Contains(t, result, "TEST_ENV")
	require.Equal(t, "TEST_ENV", result["TEST_ENV"].Name)
	require.Equal(t, "Test environment variable", result["TEST_ENV"].Description)
	require.True(t, result["TEST_ENV"].Required)
	require.Equal(t, "environment", string(result["TEST_ENV"].VariableType))

	require.Contains(t, result, "TEST_ARG")
	require.Equal(t, "TEST_ARG", result["TEST_ARG"].Name)
	require.Equal(t, "Test command argument", result["TEST_ARG"].Description)
	require.False(t, result["TEST_ARG"].Required)
	require.Equal(t, "argument", string(result["TEST_ARG"].VariableType))
}

func TestRegistry_BuildPackageResult_ValidServer(t *testing.T) {
	t.Parallel()

	logger := newTestLogger(t)
	registry, err := NewRegistry(logger, "", runtime.WithSupportedRuntimes(runtime.NPX, runtime.UVX))
	require.NoError(t, err)

	// Find a server that can actually be built
	var validServerKey string
	for key := range registry.mcpServers {
		if pkg, ok := registry.serverForID(key); ok && len(pkg.Installations) > 0 {
			validServerKey = key
			break
		}
	}
	require.NotEmpty(t, validServerKey, "Should have at least one server that can be built")

	// Test building package result
	pkg, ok := registry.serverForID(validServerKey)
	require.True(t, ok, "Should successfully build package result")
	require.Equal(t, validServerKey, pkg.Name)
	require.Equal(t, RegistryName, pkg.Source)
}

func TestRegistry_BuildPackageResult_InvalidServer(t *testing.T) {
	t.Parallel()

	logger := newTestLogger(t)
	registry, err := NewRegistry(logger, "", runtime.WithSupportedRuntimes(runtime.NPX, runtime.UVX))
	require.NoError(t, err)

	// Test with non-existent server
	_, ok := registry.serverForID("nonexistent-server")
	require.False(t, ok, "Should fail to build package result for non-existent server")
}

func TestRegistry_Arguments_ToDomainType_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		description string
		args        Arguments
		expected    packages.Arguments
	}{
		{
			name:        "empty arguments",
			description: "Empty arguments map should convert to empty result",
			args:        Arguments{},
			expected:    packages.Arguments{},
		},
		{
			name:        "all argument types",
			description: "All three argument types should be correctly converted",
			args: Arguments{
				"ENV_VAR": {
					Name:        "ENV_VAR",
					Description: "Environment variable",
					Required:    true,
					Type:        ArgumentEnv,
					Example:     "env_example",
				},
				"CLI_ARG": {
					Name:        "CLI_ARG",
					Description: "Runtime line argument",
					Required:    false,
					Type:        ArgumentValue,
					Example:     "cli_example",
				},
				"BOOL_FLAG": {
					Name:        "BOOL_FLAG",
					Description: "Boolean flag",
					Required:    false,
					Type:        ArgumentBool,
				},
			},
			expected: packages.Arguments{
				"ENV_VAR": {
					Name:         "ENV_VAR",
					Description:  "Environment variable",
					Required:     true,
					VariableType: packages.VariableTypeEnv,
					Example:      "env_example",
				},
				"CLI_ARG": {
					Name:         "CLI_ARG",
					Description:  "Runtime line argument",
					Required:     false,
					VariableType: packages.VariableTypeArg,
					Example:      "cli_example",
				},
				"BOOL_FLAG": {
					Name:         "BOOL_FLAG",
					Description:  "Boolean flag",
					Required:     false,
					VariableType: packages.VariableTypeArgBool,
					Example:      "",
				},
			},
		},
		{
			name:        "required vs optional mix",
			description: "Mix of required and optional arguments should preserve requirements",
			args: Arguments{
				"REQUIRED_ENV": {
					Name:        "REQUIRED_ENV",
					Description: "Required environment variable",
					Required:    true,
					Type:        ArgumentEnv,
					Example:     "required_value",
				},
				"OPTIONAL_ARG": {
					Name:        "OPTIONAL_ARG",
					Description: "Optional command line argument",
					Required:    false,
					Type:        ArgumentValue,
				},
			},
			expected: packages.Arguments{
				"REQUIRED_ENV": {
					Name:         "REQUIRED_ENV",
					Description:  "Required environment variable",
					Required:     true,
					VariableType: packages.VariableTypeEnv,
					Example:      "required_value",
				},
				"OPTIONAL_ARG": {
					Name:         "OPTIONAL_ARG",
					Description:  "Optional command line argument",
					Required:     false,
					VariableType: packages.VariableTypeArg,
					Example:      "",
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			t.Logf("Test description: %s", tc.description)

			result, err := tc.args.ToDomainType()
			require.NoError(t, err)
			require.Equal(t, tc.expected, result, "Argument conversion mismatch")
		})
	}
}

func TestRegistry_Arguments_ToDomainType_WithTestdata(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		testdataFile string
		serverName   string
		expectedEnv  []string
		expectedArgs []string
	}{
		{
			name:         "environment variables only",
			testdataFile: "arg_test_env_only.json",
			serverName:   "env-only-server",
			expectedEnv:  []string{"API_KEY", "DEBUG_MODE"},
		},
		{
			name:         "command-line arguments only",
			testdataFile: "arg_test_cli_only.json",
			serverName:   "cli-only-server",
			expectedArgs: []string{"--output-format", "--port", "--enable-cors"},
		},
		{
			name:         "mixed environment and arguments",
			testdataFile: "arg_test_mixed_args.json",
			serverName:   "mixed-args-server",
			expectedEnv:  []string{"DATABASE_URL"},
			expectedArgs: []string{"--config-path", "--verbose"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			registry := loadTestDataRegistry(t, tc.testdataFile)
			server, exists := registry[tc.serverName]
			require.True(t, exists)

			result, err := server.Arguments.ToDomainType()
			require.NoError(t, err)

			// Verify expected environment variables
			envVars := result.FilterBy(packages.EnvVar).Names()
			require.ElementsMatch(t, tc.expectedEnv, envVars)
			for _, key := range tc.expectedEnv {
				require.Equal(t, key, result[key].Name)
			}

			// Verify expected CLI arguments
			cliArgs := result.FilterBy(packages.Argument).Names()
			require.ElementsMatch(t, tc.expectedArgs, cliArgs)
			for _, key := range tc.expectedArgs {
				require.Equal(t, key, result[key].Name)
			}

			// Verify argument count matches expectations
			expectedTotal := len(tc.expectedEnv) + len(tc.expectedArgs)
			require.Len(t, result, expectedTotal)
		})
	}
}

func TestRegistry_BuildPackageResult_ArgumentExtraction(t *testing.T) {
	t.Parallel()

	// Create a mock registry with test data
	mockRegistry := MCPRegistry{
		"test-server": Server{
			ID:          "test-server",
			Name:        "test-server",
			DisplayName: "Test Server",
			Description: "Server for testing argument extraction",
			License:     "MIT",
			Arguments: Arguments{
				"CONFIG_FILE": {
					Name:        "CONFIG_FILE",
					Description: "Configuration file path",
					Required:    true,
					Type:        ArgumentValue,
					Example:     "/etc/config.yaml",
				},
				"API_TOKEN": {
					Name:        "API_TOKEN",
					Description: "API authentication token",
					Required:    true,
					Type:        ArgumentEnv,
					Example:     "abc123",
				},
				"DEBUG": {
					Name:        "DEBUG",
					Description: "Enable debug mode",
					Required:    false,
					Type:        ArgumentBool,
				},
			},
			Installations: map[string]Installation{
				"npx": {
					Runtime:     NPX,
					Package:     "test-server",
					Version:     "1.0.0",
					Description: "Run with npx",
					Recommended: true,
					Transports:  []string{"stdio"},
				},
			},
			Tools: Tools{
				{
					Name:        "test_tool",
					Description: "Test tool",
				},
			},
			IsOfficial: false,
		},
	}

	logger := newTestLogger(t)
	registry := &Registry{
		mcpServers:        mockRegistry,
		logger:            logger,
		supportedRuntimes: map[runtime.Runtime]struct{}{runtime.NPX: {}},
		filterOptions:     []options.Option{},
	}

	// Build package result
	pkg, ok := registry.serverForID("test-server")
	require.True(t, ok, "Should successfully build package result")
	require.Equal(t, "test-server", pkg.Name)
	require.Equal(t, "mozilla-ai", pkg.Source)

	// Verify arguments were extracted correctly
	require.Len(t, pkg.Arguments, 3, "Should have 3 arguments")

	// Check environment variable
	envVars := pkg.Arguments.FilterBy(packages.EnvVar).Names()
	require.ElementsMatch(t, []string{"API_TOKEN"}, envVars, "Should extract environment variable")

	// Check CLI arguments
	cliArgs := pkg.Arguments.FilterBy(packages.Argument).Names()
	require.ElementsMatch(t, []string{"CONFIG_FILE", "DEBUG"}, cliArgs, "Should extract CLI arguments")

	// Verify argument details
	apiToken, exists := pkg.Arguments["API_TOKEN"]
	require.True(t, exists, "API_TOKEN should exist")
	require.Equal(t, "API_TOKEN", apiToken.Name)
	require.Equal(t, packages.VariableTypeEnv, apiToken.VariableType)
	require.True(t, apiToken.Required)
	require.Equal(t, "API authentication token", apiToken.Description)
	require.Equal(t, "abc123", apiToken.Example)

	configFile, exists := pkg.Arguments["CONFIG_FILE"]
	require.True(t, exists, "CONFIG_FILE should exist")
	require.Equal(t, "CONFIG_FILE", configFile.Name)
	require.Equal(t, packages.VariableTypeArg, configFile.VariableType)
	require.True(t, configFile.Required)
	require.Equal(t, "Configuration file path", configFile.Description)
	require.Equal(t, "/etc/config.yaml", configFile.Example)

	debug, exists := pkg.Arguments["DEBUG"]
	require.True(t, exists, "DEBUG should exist")
	require.Equal(t, "DEBUG", debug.Name)
	require.Equal(t, packages.VariableTypeArgBool, debug.VariableType)
	require.False(t, debug.Required)
	require.Equal(t, "Enable debug mode", debug.Description)
}

func TestRegistry_BuildPackageResult_WithOptionalArguments(t *testing.T) {
	t.Parallel()

	testRegistry := loadTestDataRegistry(t, "registry_optional_args.json")

	tests := []struct {
		name      string
		serverID  string
		checkFunc func(t *testing.T, pkg packages.Server)
	}{
		{
			name:     "time server with optional timezone argument",
			serverID: "time",
			checkFunc: func(t *testing.T, pkg packages.Server) {
				require.Equal(t, "time", pkg.Name)
				require.Equal(t, "Time Server", pkg.DisplayName)

				// Check optional argument
				tzArg, exists := pkg.Arguments["--local-timezone"]
				require.True(t, exists)
				require.False(t, tzArg.Required)
				require.Equal(t, packages.VariableType("argument"), tzArg.VariableType)
				require.Equal(t, "America/New_York", tzArg.Example)

				// Check installation
				uvxInstall, exists := pkg.Installations[runtime.UVX]
				require.True(t, exists, "Should have UVX installation")
				require.Equal(t, "mcp-server-time", uvxInstall.Package)
			},
		},
		{
			name:     "sqlite with required argument and package field",
			serverID: "sqlite-with-package",
			checkFunc: func(t *testing.T, pkg packages.Server) {
				require.Equal(t, "sqlite-with-package", pkg.Name)

				// Check required argument
				dbArg, exists := pkg.Arguments["--db-path"]
				require.True(t, exists)
				require.True(t, dbArg.Required)

				// Check installation with package field
				uvxInstall, exists := pkg.Installations[runtime.UVX]
				require.True(t, exists, "Should have UVX installation")
				require.Equal(t, "mcp-server-sqlite", uvxInstall.Package)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			logger := newTestLogger(t)
			registry := &Registry{
				mcpServers: testRegistry,
				logger:     logger,
				supportedRuntimes: map[runtime.Runtime]struct{}{
					runtime.UVX: {},
					runtime.NPX: {},
				},
				filterOptions: []options.Option{},
			}

			pkg, ok := registry.serverForID(tc.serverID)
			require.True(t, ok, "Should successfully build package result")
			tc.checkFunc(t, pkg)
		})
	}
}
