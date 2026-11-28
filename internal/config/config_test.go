package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/require"
)

func TestInit_CreatesNewConfigFile(t *testing.T) {
	t.Parallel()

	tempFile, err := os.CreateTemp(t.TempDir(), ".mcpd.toml")
	require.NoError(t, err)
	tempFilePath := tempFile.Name()
	require.NoError(t, tempFile.Close())
	require.NoError(t, os.Remove(tempFilePath)) // Ensure file doesn't exist

	loader := &DefaultLoader{}
	err = loader.Init(tempFilePath)
	require.NoError(t, err)

	_, err = os.Stat(tempFilePath)
	require.NoError(t, err)

	content, err := os.ReadFile(tempFilePath)
	require.NoError(t, err)
	require.Equal(t, "servers = []", string(content))
}

func TestInit_ErrorsIfFileExists(t *testing.T) {
	t.Parallel()

	tempFile, err := os.CreateTemp(t.TempDir(), ".mcpd.toml")
	require.NoError(t, err)

	loader := &DefaultLoader{}
	err = loader.Init(tempFile.Name())
	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists")
}

func TestLoad_FileDoesNotExist(t *testing.T) {
	t.Parallel()

	loader := &DefaultLoader{}
	_, err := loader.Load("/nonexistent/config.toml")
	require.Error(t, err)
	require.Contains(t, err.Error(), "config file cannot be found")
}

func TestLoad_ValidConfig(t *testing.T) {
	t.Parallel()

	tempFile, err := os.CreateTemp(t.TempDir(), ".mcpd.toml")
	require.NoError(t, err)

	content := `[[servers]]
name = "test"
package = "x::test@latest"
`
	require.NoError(t, os.WriteFile(tempFile.Name(), []byte(content), 0o644))

	loader := &DefaultLoader{}
	cfg, err := loader.Load(tempFile.Name())
	require.NoError(t, err)
	require.Len(t, cfg.ListServers(), 1)

	server := cfg.ListServers()[0]
	require.Equal(t, "test", server.Name)
	require.Equal(t, "x::test@latest", server.Package)
}

func TestAddServer_AppendsServerAndPersists(t *testing.T) {
	tempFile, err := os.CreateTemp(t.TempDir(), ".mcpd.toml")
	require.NoError(t, err)
	require.NoError(t, tempFile.Close())

	cfg := &Config{
		configFilePath: tempFile.Name(),
	}

	entry := ServerEntry{
		Name:    "my-server",
		Package: "x::my-server@latest",
		Tools:   []string{"tool1"},
	}

	err = cfg.AddServer(entry)
	require.NoError(t, err)

	var loaded Config
	_, err = toml.DecodeFile(tempFile.Name(), &loaded)
	require.NoError(t, err)

	require.Len(t, loaded.Servers, 1)
	require.Equal(t, entry, loaded.Servers[0])
}

func TestRemoveServer_RemovesCorrectEntry(t *testing.T) {
	tempFile, err := os.CreateTemp(t.TempDir(), ".mcpd.toml")
	require.NoError(t, err)
	require.NoError(t, tempFile.Close())

	cfg := &Config{
		configFilePath: tempFile.Name(),
		Servers: []ServerEntry{
			{Name: "foo", Package: "x::foo@latest"},
			{Name: "bar", Package: "x::bar@latest"},
		},
	}

	require.NoError(t, cfg.saveConfig())

	err = cfg.RemoveServer("foo")
	require.NoError(t, err)

	var loaded Config
	_, err = toml.DecodeFile(tempFile.Name(), &loaded)
	require.NoError(t, err)

	require.Len(t, loaded.Servers, 1)
	require.Equal(t, "bar", loaded.Servers[0].Name)
}

func TestRemoveServer_MatchesWithoutRegardToCase(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		configFilePath: filepath.Join(t.TempDir(), ".mcpd.toml"),
		Servers: []ServerEntry{
			{Name: "github", Package: "x::github@latest"},
			{Name: "time", Package: "x::time@latest"},
		},
	}

	require.NoError(t, cfg.RemoveServer(" GitHub "))

	require.Len(t, cfg.Servers, 1)
	require.Equal(t, "time", cfg.Servers[0].Name)
}

// TestRemoveServer_DecidesBetweenNamesSharingOneKey pins that a configuration written by an earlier
// release, which may hold two names sharing one key, can still be repaired one entry at a time.
func TestRemoveServer_DecidesBetweenNamesSharingOneKey(t *testing.T) {
	t.Parallel()

	newConfig := func(t *testing.T) *Config {
		t.Helper()

		return &Config{
			configFilePath: filepath.Join(t.TempDir(), ".mcpd.toml"),
			Servers: []ServerEntry{
				{Name: "github", Package: "x::github@latest"},
				{Name: "GitHub", Package: "x::other@latest"},
			},
		}
	}

	t.Run("an exact name removes that entry alone", func(t *testing.T) {
		t.Parallel()

		cfg := newConfig(t)
		require.NoError(t, cfg.RemoveServer("GitHub"))

		require.Len(t, cfg.Servers, 1)
		require.Equal(t, "github", cfg.Servers[0].Name)
	})

	t.Run("a name matching neither exactly names both", func(t *testing.T) {
		t.Parallel()

		cfg := newConfig(t)
		err := cfg.RemoveServer("GITHUB")

		require.Error(t, err)
		require.Contains(t, err.Error(), "GitHub")
		require.Contains(t, err.Error(), "github")
		require.Len(t, cfg.Servers, 2, "an ambiguous name must not remove anything")
	})
}

func TestRemoveServer_ErrorsIfNotFound(t *testing.T) {
	cfg := &Config{
		configFilePath: "dummy-path",
		Servers: []ServerEntry{
			{Name: "only-server", Package: "x::only-server@latest"},
		},
	}

	err := cfg.RemoveServer("not-there")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestRemoveServer_ErrorsIfEmptyName(t *testing.T) {
	cfg := &Config{
		configFilePath: "dummy-path",
	}

	err := cfg.RemoveServer("  ")
	require.Error(t, err)
	require.Contains(t, err.Error(), "server name cannot be empty")
}

func TestStripVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "full package name",
			input:    "docker::greptime/greptimedb@latest",
			expected: "docker::greptime/greptimedb",
		},
		{
			name:     "missing version",
			input:    "docker::greptime/greptimedb",
			expected: "docker::greptime/greptimedb",
		},
		{
			name:     "missing prefix",
			input:    "something@v1.0.0",
			expected: "something",
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

			require.Equal(t, tc.expected, stripVersion(tc.input))
		})
	}
}

func TestStripPrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "missing version",
			input:    "docker::greptime/greptimedb",
			expected: "greptime/greptimedb",
		},
		{
			name:     "multiple prefix",
			input:    "foo::bar::baz",
			expected: "bar::baz",
		},
		{
			name:     "no prefix",
			input:    "greptime/greptimedb",
			expected: "greptime/greptimedb",
		},
		{
			name:     "only prefix (no runtime)",
			input:    "::greptime/greptimedb",
			expected: "greptime/greptimedb",
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

			require.Equal(t, tc.expected, stripPrefix(tc.input))
		})
	}
}

func TestServerEntry_PackageVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		entry    ServerEntry
		expected string
	}{
		{
			name: "with prefix and version",
			entry: ServerEntry{
				Package: "docker::greptime/greptimedb@latest",
			},
			expected: "latest",
		},
		{
			name: "with prefix but no version",
			entry: ServerEntry{
				Package: "docker::greptime/greptimedb",
			},
			expected: "greptime/greptimedb",
		},
		{
			name: "no prefix but with version",
			entry: ServerEntry{
				Package: "greptime/greptimedb@v1.2.3",
			},
			expected: "v1.2.3",
		},
		{
			name: "empty package",
			entry: ServerEntry{
				Package: "",
			},
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.expected, tc.entry.PackageVersion())
		})
	}
}

func TestServerEntry_PackageName(t *testing.T) {
	tests := []struct {
		name     string
		entry    ServerEntry
		expected string
	}{
		{
			name: "with prefix and version",
			entry: ServerEntry{
				Package: "docker::greptime/greptimedb@latest",
			},
			expected: "greptime/greptimedb",
		},
		{
			name: "with prefix and no version",
			entry: ServerEntry{
				Package: "docker::greptime/greptimedb",
			},
			expected: "greptime/greptimedb",
		},
		{
			name: "no prefix, with version",
			entry: ServerEntry{
				Package: "greptime/greptimedb@v2.0.0",
			},
			expected: "greptime/greptimedb",
		},
		{
			name: "no prefix, no version",
			entry: ServerEntry{
				Package: "greptime/greptimedb",
			},
			expected: "greptime/greptimedb",
		},
		{
			name: "only prefix",
			entry: ServerEntry{
				Package: "::greptime/greptimedb",
			},
			expected: "greptime/greptimedb",
		},
		{
			name: "empty package",
			entry: ServerEntry{
				Package: "",
			},
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.expected, tc.entry.PackageName())
		})
	}
}

func TestUpsertPlugin_CreatesAndPersists(t *testing.T) {
	t.Parallel()

	tempFile, err := os.CreateTemp(t.TempDir(), ".mcpd.toml")
	require.NoError(t, err)

	cfg := &Config{
		Servers:        []ServerEntry{{Name: "test", Package: "x::test@latest"}},
		configFilePath: tempFile.Name(),
	}

	entry := PluginEntry{
		Name:       "jwt-auth",
		CommitHash: testStringPtr(t, "abc123"),
		Required:   testBoolPtr(t, true),
		Flows:      []Flow{FlowRequest, FlowResponse},
	}

	result, err := cfg.UpsertPlugin(CategoryAuthentication, entry)
	require.NoError(t, err)
	require.Equal(t, "created", string(result))

	// Verify saved to disk.
	var loaded Config
	_, err = toml.DecodeFile(tempFile.Name(), &loaded)
	require.NoError(t, err)

	require.NotNil(t, loaded.Plugins)
	require.Len(t, loaded.Plugins.Authentication, 1)
	require.Equal(t, "jwt-auth", loaded.Plugins.Authentication[0].Name)
	require.Equal(t, "abc123", *loaded.Plugins.Authentication[0].CommitHash)
	require.True(t, *loaded.Plugins.Authentication[0].Required)
	require.Equal(t, []Flow{FlowRequest, FlowResponse}, loaded.Plugins.Authentication[0].Flows)
}

func TestUpsertPlugin_UpdatesExisting(t *testing.T) {
	t.Parallel()

	tempFile, err := os.CreateTemp(t.TempDir(), ".mcpd.toml")
	require.NoError(t, err)

	cfg := &Config{
		Servers: []ServerEntry{{Name: "test", Package: "x::test@latest"}},
		Plugins: &PluginConfig{
			Authentication: []PluginEntry{
				{Name: "jwt-auth", Flows: []Flow{FlowRequest}},
			},
		},
		configFilePath: tempFile.Name(),
	}

	// Save initial state.
	require.NoError(t, cfg.saveConfig())

	// Update plugin.
	entry := PluginEntry{
		Name:  "jwt-auth",
		Flows: []Flow{FlowRequest, FlowResponse},
	}

	result, err := cfg.UpsertPlugin(CategoryAuthentication, entry)
	require.NoError(t, err)
	require.Equal(t, "updated", string(result))

	// Verify saved to disk.
	var loaded Config
	_, err = toml.DecodeFile(tempFile.Name(), &loaded)
	require.NoError(t, err)

	require.Len(t, loaded.Plugins.Authentication, 1)
	require.Equal(t, []Flow{FlowRequest, FlowResponse}, loaded.Plugins.Authentication[0].Flows)
}

func TestDeletePlugin_RemovesAndPersists(t *testing.T) {
	t.Parallel()

	tempFile, err := os.CreateTemp(t.TempDir(), ".mcpd.toml")
	require.NoError(t, err)

	cfg := &Config{
		Servers: []ServerEntry{{Name: "test", Package: "x::test@latest"}},
		Plugins: &PluginConfig{
			Authentication: []PluginEntry{
				{Name: "jwt-auth", Flows: []Flow{FlowRequest}},
				{Name: "oauth2", Flows: []Flow{FlowRequest}},
			},
		},
		configFilePath: tempFile.Name(),
	}

	require.NoError(t, cfg.saveConfig())

	result, err := cfg.DeletePlugin(CategoryAuthentication, "jwt-auth")
	require.NoError(t, err)
	require.Equal(t, "deleted", string(result))

	// Verify saved to disk.
	var loaded Config
	_, err = toml.DecodeFile(tempFile.Name(), &loaded)
	require.NoError(t, err)

	require.Len(t, loaded.Plugins.Authentication, 1)
	require.Equal(t, "oauth2", loaded.Plugins.Authentication[0].Name)
}

func TestLoad_ValidConfigWithPlugins(t *testing.T) {
	t.Parallel()

	tempFile, err := os.CreateTemp(t.TempDir(), ".mcpd.toml")
	require.NoError(t, err)

	content := `[[servers]]
name = "test"
package = "x::test@latest"

[[plugins.authentication]]
name = "jwt-auth"
commit_hash = "abc123"
required = true
flows = ["request", "response"]

[[plugins.observability]]
name = "metrics"
flows = ["request"]
`
	require.NoError(t, os.WriteFile(tempFile.Name(), []byte(content), 0o644))

	loader := &DefaultLoader{}
	cfg, err := loader.Load(tempFile.Name())
	require.NoError(t, err)

	// Verify server loaded.
	require.Len(t, cfg.ListServers(), 1)

	// Type assert to access plugin methods.
	pluginCfg, ok := cfg.(*Config)
	require.True(t, ok, "Config should support plugin operations")

	// Verify plugins loaded.
	authPlugins := pluginCfg.Plugins.ListPlugins(CategoryAuthentication)
	require.Len(t, authPlugins, 1)
	require.Equal(t, "jwt-auth", authPlugins[0].Name)
	require.Equal(t, "abc123", *authPlugins[0].CommitHash)
	require.True(t, *authPlugins[0].Required)
	require.Equal(t, []Flow{FlowRequest, FlowResponse}, authPlugins[0].Flows)

	obsPlugins := pluginCfg.Plugins.ListPlugins(CategoryObservability)
	require.Len(t, obsPlugins, 1)
	require.Equal(t, "metrics", obsPlugins[0].Name)
	require.Equal(t, []Flow{FlowRequest}, obsPlugins[0].Flows)
}

func TestLoad_InvalidPluginConfigFails(t *testing.T) {
	t.Parallel()

	tempFile, err := os.CreateTemp(t.TempDir(), ".mcpd.toml")
	require.NoError(t, err)

	content := `[[servers]]
name = "test"
package = "x::test@latest"

[[plugins.authentication]]
name = ""
flows = ["request"]
`
	require.NoError(t, os.WriteFile(tempFile.Name(), []byte(content), 0o644))

	loader := &DefaultLoader{}
	_, err = loader.Load(tempFile.Name())
	require.Error(t, err)
	require.Contains(t, err.Error(), "plugin configuration error")
}

func TestLoad_StaticTestdata_BasicPlugins(t *testing.T) {
	t.Parallel()

	loader := &DefaultLoader{}
	cfg, err := loader.Load("testdata/basic_plugins.toml")
	require.NoError(t, err)

	servers := cfg.ListServers()
	require.Len(t, servers, 1)
	require.Equal(t, "test-server", servers[0].Name)
	require.Equal(t, "uvx::test-package@latest", servers[0].Package)
	require.Equal(t, []string{"test_tool"}, servers[0].Tools)

	pluginCfg, ok := cfg.(*Config)
	require.True(t, ok)

	authPlugins := pluginCfg.Plugins.ListPlugins(CategoryAuthentication)
	require.Len(t, authPlugins, 1)
	require.Equal(t, "jwt-auth", authPlugins[0].Name)
	require.NotNil(t, authPlugins[0].CommitHash)
	require.Equal(t, "abc123", *authPlugins[0].CommitHash)
	require.NotNil(t, authPlugins[0].Required)
	require.True(t, *authPlugins[0].Required)
	require.Equal(t, []Flow{FlowRequest}, authPlugins[0].Flows)
}

func TestLoad_StaticTestdata_MultiplePlugins(t *testing.T) {
	t.Parallel()

	loader := &DefaultLoader{}
	cfg, err := loader.Load("testdata/multiple_plugins.toml")
	require.NoError(t, err)

	servers := cfg.ListServers()
	require.Len(t, servers, 1)
	require.Equal(t, "api-server", servers[0].Name)

	pluginCfg, ok := cfg.(*Config)
	require.True(t, ok)

	authPlugins := pluginCfg.Plugins.ListPlugins(CategoryAuthentication)
	require.Len(t, authPlugins, 2)
	require.Equal(t, "jwt-auth", authPlugins[0].Name)
	require.Equal(t, "api-key-auth", authPlugins[1].Name)

	authzPlugins := pluginCfg.Plugins.ListPlugins(CategoryAuthorization)
	require.Len(t, authzPlugins, 1)
	require.Equal(t, "rbac", authzPlugins[0].Name)
	require.True(t, *authzPlugins[0].Required)

	rateLimitPlugins := pluginCfg.Plugins.ListPlugins(CategoryRateLimiting)
	require.Len(t, rateLimitPlugins, 1)
	require.Equal(t, "token-bucket", rateLimitPlugins[0].Name)

	obsPlugins := pluginCfg.Plugins.ListPlugins(CategoryObservability)
	require.Len(t, obsPlugins, 1)
	require.Equal(t, "metrics", obsPlugins[0].Name)
	require.Equal(t, []Flow{FlowRequest, FlowResponse}, obsPlugins[0].Flows)

	auditPlugins := pluginCfg.Plugins.ListPlugins(CategoryAudit)
	require.Len(t, auditPlugins, 1)
	require.Equal(t, "compliance-logger", auditPlugins[0].Name)
	require.Equal(t, []Flow{FlowResponse}, auditPlugins[0].Flows)
}

func TestLoad_StaticTestdata_MinimalPlugins(t *testing.T) {
	t.Parallel()

	loader := &DefaultLoader{}
	cfg, err := loader.Load("testdata/minimal_plugins.toml")
	require.NoError(t, err)

	servers := cfg.ListServers()
	require.Len(t, servers, 1)
	require.Equal(t, "simple-server", servers[0].Name)

	pluginCfg, ok := cfg.(*Config)
	require.True(t, ok)

	require.Len(t, pluginCfg.Plugins.ListPlugins(CategoryAuthentication), 0)
	require.Len(t, pluginCfg.Plugins.ListPlugins(CategoryAuthorization), 0)
	require.Len(t, pluginCfg.Plugins.ListPlugins(CategoryRateLimiting), 0)
	require.Len(t, pluginCfg.Plugins.ListPlugins(CategoryValidation), 0)
	require.Len(t, pluginCfg.Plugins.ListPlugins(CategoryContent), 0)
	require.Len(t, pluginCfg.Plugins.ListPlugins(CategoryObservability), 0)
	require.Len(t, pluginCfg.Plugins.ListPlugins(CategoryAudit), 0)
}

func TestLoad_StaticTestdata_InvalidPlugins(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		testdataFile  string
		expectedError string
	}{
		{
			name:          "empty plugin name",
			testdataFile:  "testdata/invalid_empty_plugin_name.toml",
			expectedError: "plugin name is required",
		},
		{
			name:          "no flows",
			testdataFile:  "testdata/invalid_no_flows.toml",
			expectedError: "at least one flow is required",
		},
		{
			name:          "invalid flow value",
			testdataFile:  "testdata/invalid_bad_flow.toml",
			expectedError: "invalid flow",
		},
		{
			name:          "duplicate flows",
			testdataFile:  "testdata/invalid_duplicate_flows.toml",
			expectedError: "duplicate flow",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			loader := &DefaultLoader{}
			_, err := loader.Load(tc.testdataFile)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.expectedError)
		})
	}
}

func TestLoadSaveRoundTrip_Plugins(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, ".mcpd.toml")

	sourceData, err := os.ReadFile("testdata/multiple_plugins.toml")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(tempFile, sourceData, 0o644))

	loader := &DefaultLoader{}
	cfg, err := loader.Load(tempFile)
	require.NoError(t, err)

	config, ok := cfg.(*Config)
	require.True(t, ok)

	require.NoError(t, config.SaveConfig())

	reloaded, err := loader.Load(tempFile)
	require.NoError(t, err)

	reloadedConfig, ok := reloaded.(*Config)
	require.True(t, ok)

	authPlugins := reloadedConfig.Plugins.ListPlugins(CategoryAuthentication)
	require.Len(t, authPlugins, 2)

	authzPlugins := reloadedConfig.Plugins.ListPlugins(CategoryAuthorization)
	require.Len(t, authzPlugins, 1)

	rateLimitPlugins := reloadedConfig.Plugins.ListPlugins(CategoryRateLimiting)
	require.Len(t, rateLimitPlugins, 1)

	obsPlugins := reloadedConfig.Plugins.ListPlugins(CategoryObservability)
	require.Len(t, obsPlugins, 1)

	auditPlugins := reloadedConfig.Plugins.ListPlugins(CategoryAudit)
	require.Len(t, auditPlugins, 1)

	require.Equal(t, "jwt-auth", authPlugins[0].Name)
	require.NotNil(t, authPlugins[0].CommitHash)
	require.Equal(t, "abc123", *authPlugins[0].CommitHash)
	require.True(t, *authPlugins[0].Required)

	require.Equal(t, "metrics", obsPlugins[0].Name)
	require.Equal(t, []Flow{FlowRequest, FlowResponse}, obsPlugins[0].Flows)
}

func TestMovePlugin_ToCategory(t *testing.T) {
	t.Parallel()

	tempFile, err := os.CreateTemp(t.TempDir(), ".mcpd.toml")
	require.NoError(t, err)

	cfg := &Config{
		Servers: []ServerEntry{{Name: "test", Package: "x::test@latest"}},
		Plugins: &PluginConfig{
			Authentication: []PluginEntry{
				{Name: "jwt-auth", Flows: []Flow{FlowRequest}},
			},
		},
		configFilePath: tempFile.Name(),
	}

	require.NoError(t, cfg.saveConfig())

	result, err := cfg.MovePlugin(CategoryAuthentication, "jwt-auth", WithToCategory(CategoryAudit))
	require.NoError(t, err)
	require.Equal(t, "updated", string(result))

	// Verify persisted.
	var loaded Config
	_, err = toml.DecodeFile(tempFile.Name(), &loaded)
	require.NoError(t, err)

	require.Empty(t, loaded.Plugins.Authentication)
	require.Len(t, loaded.Plugins.Audit, 1)
	require.Equal(t, "jwt-auth", loaded.Plugins.Audit[0].Name)
}

func TestMovePlugin_ToPosition(t *testing.T) {
	t.Parallel()

	tempFile, err := os.CreateTemp(t.TempDir(), ".mcpd.toml")
	require.NoError(t, err)

	cfg := &Config{
		Servers: []ServerEntry{{Name: "test", Package: "x::test@latest"}},
		Plugins: &PluginConfig{
			Authentication: []PluginEntry{
				{Name: "a", Flows: []Flow{FlowRequest}},
				{Name: "b", Flows: []Flow{FlowRequest}},
				{Name: "c", Flows: []Flow{FlowRequest}},
			},
		},
		configFilePath: tempFile.Name(),
	}

	require.NoError(t, cfg.saveConfig())

	result, err := cfg.MovePlugin(CategoryAuthentication, "c", WithPosition(1))
	require.NoError(t, err)
	require.Equal(t, "updated", string(result))

	plugins := cfg.Plugins.ListPlugins(CategoryAuthentication)
	require.Equal(t, "c", plugins[0].Name)
	require.Equal(t, "a", plugins[1].Name)
	require.Equal(t, "b", plugins[2].Name)
}

func TestMovePlugin_Before(t *testing.T) {
	t.Parallel()

	tempFile, err := os.CreateTemp(t.TempDir(), ".mcpd.toml")
	require.NoError(t, err)

	cfg := &Config{
		Servers: []ServerEntry{{Name: "test", Package: "x::test@latest"}},
		Plugins: &PluginConfig{
			Authentication: []PluginEntry{
				{Name: "a", Flows: []Flow{FlowRequest}},
				{Name: "b", Flows: []Flow{FlowRequest}},
			},
		},
		configFilePath: tempFile.Name(),
	}

	require.NoError(t, cfg.saveConfig())

	result, err := cfg.MovePlugin(CategoryAuthentication, "b", WithBefore("a"))
	require.NoError(t, err)
	require.Equal(t, "updated", string(result))

	plugins := cfg.Plugins.ListPlugins(CategoryAuthentication)
	require.Equal(t, "b", plugins[0].Name)
	require.Equal(t, "a", plugins[1].Name)
}

func TestMovePlugin_After(t *testing.T) {
	t.Parallel()

	tempFile, err := os.CreateTemp(t.TempDir(), ".mcpd.toml")
	require.NoError(t, err)

	cfg := &Config{
		Servers: []ServerEntry{{Name: "test", Package: "x::test@latest"}},
		Plugins: &PluginConfig{
			Authentication: []PluginEntry{
				{Name: "a", Flows: []Flow{FlowRequest}},
				{Name: "b", Flows: []Flow{FlowRequest}},
				{Name: "c", Flows: []Flow{FlowRequest}},
			},
		},
		configFilePath: tempFile.Name(),
	}

	require.NoError(t, cfg.saveConfig())

	result, err := cfg.MovePlugin(CategoryAuthentication, "a", WithAfter("b"))
	require.NoError(t, err)
	require.Equal(t, "updated", string(result))

	plugins := cfg.Plugins.ListPlugins(CategoryAuthentication)
	require.Equal(t, "b", plugins[0].Name)
	require.Equal(t, "a", plugins[1].Name)
	require.Equal(t, "c", plugins[2].Name)
}

func TestMovePlugin_ToCategoryAndPosition(t *testing.T) {
	t.Parallel()

	tempFile, err := os.CreateTemp(t.TempDir(), ".mcpd.toml")
	require.NoError(t, err)

	cfg := &Config{
		Servers: []ServerEntry{{Name: "test", Package: "x::test@latest"}},
		Plugins: &PluginConfig{
			Authentication: []PluginEntry{
				{Name: "jwt-auth", Flows: []Flow{FlowRequest}},
			},
			Audit: []PluginEntry{
				{Name: "audit-a", Flows: []Flow{FlowRequest}},
				{Name: "audit-b", Flows: []Flow{FlowRequest}},
			},
		},
		configFilePath: tempFile.Name(),
	}

	require.NoError(t, cfg.saveConfig())

	result, err := cfg.MovePlugin(
		CategoryAuthentication,
		"jwt-auth",
		WithToCategory(CategoryAudit),
		WithPosition(1),
	)
	require.NoError(t, err)
	require.Equal(t, "updated", string(result))

	// Verify moved to correct position.
	authPlugins := cfg.Plugins.ListPlugins(CategoryAuthentication)
	require.Empty(t, authPlugins)

	auditPlugins := cfg.Plugins.ListPlugins(CategoryAudit)
	require.Equal(t, "jwt-auth", auditPlugins[0].Name)
	require.Equal(t, "audit-a", auditPlugins[1].Name)
	require.Equal(t, "audit-b", auditPlugins[2].Name)
}

func TestMovePlugin_ToCategoryWithNoopPosition_Persists(t *testing.T) {
	t.Parallel()

	tempFile, err := os.CreateTemp(t.TempDir(), ".mcpd.toml")
	require.NoError(t, err)

	cfg := &Config{
		Servers: []ServerEntry{{Name: "test", Package: "x::test@latest"}},
		Plugins: &PluginConfig{
			Authentication: []PluginEntry{
				{Name: "jwt-auth", Flows: []Flow{FlowRequest}},
			},
		},
		configFilePath: tempFile.Name(),
	}

	require.NoError(t, cfg.saveConfig())

	// The target category is empty, so the positional step is a no-op
	// (single-element slice) — but the category move must still persist.
	result, err := cfg.MovePlugin(
		CategoryAuthentication,
		"jwt-auth",
		WithToCategory(CategoryAudit),
		WithPosition(1),
	)
	require.NoError(t, err)
	require.Equal(t, "updated", string(result))

	// Verify persisted to disk, not just in memory.
	var loaded Config
	_, err = toml.DecodeFile(tempFile.Name(), &loaded)
	require.NoError(t, err)

	require.Empty(t, loaded.Plugins.Authentication)
	require.Len(t, loaded.Plugins.Audit, 1)
	require.Equal(t, "jwt-auth", loaded.Plugins.Audit[0].Name)
}

func TestMovePlugin_ToCategoryWithInvalidPosition_NotPersisted(t *testing.T) {
	t.Parallel()

	tempFile, err := os.CreateTemp(t.TempDir(), ".mcpd.toml")
	require.NoError(t, err)

	cfg := &Config{
		Servers: []ServerEntry{{Name: "test", Package: "x::test@latest"}},
		Plugins: &PluginConfig{
			Authentication: []PluginEntry{
				{Name: "jwt-auth", Flows: []Flow{FlowRequest}},
			},
		},
		configFilePath: tempFile.Name(),
	}

	require.NoError(t, cfg.saveConfig())

	// The plugin lands in the empty target category as its only entry, so the
	// positional step cannot reorder it — but an invalid position must still be
	// rejected rather than silently accepted, and the category move rolled back.
	result, err := cfg.MovePlugin(
		CategoryAuthentication,
		"jwt-auth",
		WithToCategory(CategoryAudit),
		WithPosition(0),
	)
	require.ErrorContains(t, err, "invalid position")
	require.Equal(t, "noop", string(result))

	// In-memory state is rolled back to the original category.
	require.Len(t, cfg.Plugins.Authentication, 1)
	require.Equal(t, "jwt-auth", cfg.Plugins.Authentication[0].Name)
	require.Empty(t, cfg.Plugins.Audit)

	// Nothing was persisted: the plugin remains in its original category on disk.
	var loaded Config
	_, err = toml.DecodeFile(tempFile.Name(), &loaded)
	require.NoError(t, err)

	require.Len(t, loaded.Plugins.Authentication, 1)
	require.Equal(t, "jwt-auth", loaded.Plugins.Authentication[0].Name)
	require.Empty(t, loaded.Plugins.Audit)
}

func TestMovePlugin_NoPluginsConfigured(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		configFilePath: t.TempDir() + "/test.toml",
	}

	result, err := cfg.MovePlugin(CategoryAuthentication, "jwt-auth", WithToCategory(CategoryAudit))
	require.ErrorContains(t, err, "no plugins configured")
	require.Equal(t, "noop", string(result))
}

func TestMovePlugin_PluginNotFound(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		configFilePath: t.TempDir() + "/test.toml",
		Plugins: &PluginConfig{
			Authentication: []PluginEntry{
				{Name: "other", Flows: []Flow{FlowRequest}},
			},
		},
	}

	result, err := cfg.MovePlugin(CategoryAuthentication, "nonexistent", WithToCategory(CategoryAudit))
	require.ErrorContains(t, err, "not found")
	require.Equal(t, "noop", string(result))
}

func TestMovePlugin_NoopWhenAlreadyInPosition(t *testing.T) {
	t.Parallel()

	tempFile, err := os.CreateTemp(t.TempDir(), ".mcpd.toml")
	require.NoError(t, err)

	cfg := &Config{
		Servers: []ServerEntry{{Name: "test", Package: "x::test@latest"}},
		Plugins: &PluginConfig{
			Authentication: []PluginEntry{
				{Name: "a", Flows: []Flow{FlowRequest}},
				{Name: "b", Flows: []Flow{FlowRequest}},
			},
		},
		configFilePath: tempFile.Name(),
	}

	require.NoError(t, cfg.saveConfig())

	result, err := cfg.MovePlugin(CategoryAuthentication, "a", WithPosition(1))
	require.NoError(t, err)
	require.Equal(t, "noop", string(result))
}

func TestMovePlugin_ForceOverwriteExisting(t *testing.T) {
	t.Parallel()

	tempFile, err := os.CreateTemp(t.TempDir(), ".mcpd.toml")
	require.NoError(t, err)

	cfg := &Config{
		Servers: []ServerEntry{{Name: "test", Package: "x::test@latest"}},
		Plugins: &PluginConfig{
			Authentication: []PluginEntry{
				{Name: "shared", Flows: []Flow{FlowRequest}},
			},
			Audit: []PluginEntry{
				{Name: "shared", Flows: []Flow{FlowResponse}},
			},
		},
		configFilePath: tempFile.Name(),
	}

	require.NoError(t, cfg.saveConfig())

	result, err := cfg.MovePlugin(
		CategoryAuthentication,
		"shared",
		WithToCategory(CategoryAudit),
		WithForce(true),
	)
	require.NoError(t, err)
	require.Equal(t, "updated", string(result))

	// Verify: auth empty, audit has plugin with FlowRequest.
	authPlugins := cfg.Plugins.ListPlugins(CategoryAuthentication)
	require.Empty(t, authPlugins)

	auditPlugins := cfg.Plugins.ListPlugins(CategoryAudit)
	require.Len(t, auditPlugins, 1)
	require.Equal(t, []Flow{FlowRequest}, auditPlugins[0].Flows)
}

func TestMovePlugin_ErrorWithoutForceWhenDuplicateExists(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		configFilePath: t.TempDir() + "/test.toml",
		Plugins: &PluginConfig{
			Authentication: []PluginEntry{
				{Name: "shared", Flows: []Flow{FlowRequest}},
			},
			Audit: []PluginEntry{
				{Name: "shared", Flows: []Flow{FlowResponse}},
			},
		},
	}

	result, err := cfg.MovePlugin(CategoryAuthentication, "shared", WithToCategory(CategoryAudit))
	require.ErrorContains(t, err, "already exists")
	require.Equal(t, "noop", string(result))
}

func TestSaveConfig_RefusesAConfigItCannotLoad(t *testing.T) {
	t.Parallel()

	path := t.TempDir() + "/.mcpd.toml"
	cfg := &Config{
		configFilePath: path,
		Daemon: &DaemonConfig{
			API: &APIConfigSection{
				CORS: &CORSConfigSection{
					Enable: testBoolPtr(t, true),
				},
			},
		},
	}

	err := cfg.saveConfig()
	require.Error(t, err)
	require.Contains(t, err.Error(), "CORS is enabled but no origins are allowed")
	require.NoFileExists(t, path, "an invalid config must not reach disk")
}

// TestAddServer_RefusesANameDifferingOnlyByCase pins that a configuration cannot gain two servers that
// resolve to one server at runtime. Lookups match a name without regard to case, so such a pair collapses
// onto one entry, leaving one server unreachable and its process unmanaged.
func TestAddServer_RefusesANameDifferingOnlyByCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		existing  []string
		candidate string
		expectErr string
	}{
		{
			name:      "an exact duplicate",
			existing:  []string{"github"},
			candidate: "github",
			expectErr: "duplicate server name",
		},
		{
			name:      "differing only by case",
			existing:  []string{"github"},
			candidate: "GitHub",
			expectErr: "duplicate server name",
		},
		{
			name:      "differing only by case and whitespace",
			existing:  []string{"GitHub"},
			candidate: "  GITHUB  ",
			expectErr: "duplicate server name",
		},
		{
			name:      "genuinely distinct names",
			existing:  []string{"GitHub", "GitLab"},
			candidate: "time",
			expectErr: "",
		},
		{
			name:      "no existing servers",
			existing:  nil,
			candidate: "GitHub",
			expectErr: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := &Config{configFilePath: filepath.Join(t.TempDir(), ".mcpd.toml")}
			for _, name := range tc.existing {
				cfg.Servers = append(cfg.Servers, ServerEntry{
					Name:    name,
					Package: "uvx::example@1.0.0",
				})
			}

			err := cfg.AddServer(ServerEntry{
				Name:    tc.candidate,
				Package: "uvx::other@1.0.0",
			})

			if tc.expectErr == "" {
				require.NoError(t, err)
				require.Len(t, cfg.Servers, len(tc.existing)+1)

				return
			}

			require.Error(t, err)
			require.Contains(t, err.Error(), tc.expectErr)
			require.Len(t, cfg.Servers, len(tc.existing), "a refused entry must not reach the configuration")
		})
	}
}

// TestLoad_AcceptsNamesDifferingOnlyByCase pins that such a configuration can still be read. Every command
// loads the configuration first, so refusing here would leave no command able to repair the pair.
func TestLoad_AcceptsNamesDifferingOnlyByCase(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), ".mcpd.toml")
	content := `[[servers]]
name = "github"
package = "uvx::example@1.0.0"

[[servers]]
name = "GitHub"
package = "uvx::other@1.0.0"
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	cfg, err := (&DefaultLoader{}).Load(path)
	require.NoError(t, err)
	require.Len(t, cfg.ListServers(), 2)

	require.NoError(t, cfg.RemoveServer("GitHub"))
	require.Len(t, cfg.ListServers(), 1)
}

// TestLoad_RefusesOneServerNameHeldTwice pins that a configuration holding one server name twice is
// refused, because every lookup would reach only the first of the two entries.
func TestLoad_RefusesOneServerNameHeldTwice(t *testing.T) {
	t.Parallel()

	tempFile := filepath.Join(t.TempDir(), ".mcpd.toml")
	content := `
[[servers]]
  name = "github"
  package = "uvx::official-github@1.0.0"
  tools = ["search_issues"]

[[servers]]
  name = "github"
  package = "uvx::other-github@9.9.9"
  tools = ["run_command"]
`
	require.NoError(t, os.WriteFile(tempFile, []byte(content), 0o600))

	_, err := (&DefaultLoader{}).Load(tempFile)
	require.Error(t, err)
	require.Contains(t, err.Error(), "duplicate server name 'github'")
}

// TestRemoveServer_RemovesOneEntry pins that removing a server takes out the entry that was named and
// no other, so a name held twice cannot take both entries with it.
func TestRemoveServer_RemovesOneEntry(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		configFilePath: filepath.Join(t.TempDir(), ".mcpd.toml"),
		Servers: []ServerEntry{
			{Name: "github", Package: "x::official@1.0.0"},
			{Name: "github", Package: "x::other@9.9.9"},
		},
	}

	require.NoError(t, cfg.RemoveServer("github"))

	require.Len(t, cfg.Servers, 1)
}

// TestUpdateServer_ReplacesTheEntryAndPersists pins that updating a server writes the replacement to the
// configuration file.
func TestUpdateServer_ReplacesTheEntryAndPersists(t *testing.T) {
	t.Parallel()

	tempFile := filepath.Join(t.TempDir(), ".mcpd.toml")
	cfg := &Config{
		configFilePath: tempFile,
		Servers: []ServerEntry{
			{Name: "github", Package: "x::github@1.0.0", Tools: []string{"search_issues"}},
			{Name: "time", Package: "x::time@1.0.0", Tools: []string{"get_time"}},
		},
	}

	updated := ServerEntry{Name: "github", Package: "x::github@1.0.0", Tools: []string{"list_repos"}}
	require.NoError(t, cfg.UpdateServer(updated))

	var loaded Config
	_, err := toml.DecodeFile(tempFile, &loaded)
	require.NoError(t, err)

	require.Len(t, loaded.Servers, 2)
	require.Equal(t, updated, loaded.Servers[0])
	require.Equal(t, "time", loaded.Servers[1].Name)
}

// TestUpdateServer_KeepsAnEntrySharingOneNameKey pins that updating one of two entries whose names share
// a key leaves the other in place.
func TestUpdateServer_KeepsAnEntrySharingOneNameKey(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		configFilePath: filepath.Join(t.TempDir(), ".mcpd.toml"),
		Servers: []ServerEntry{
			{Name: "GitHub", Package: "x::official@1.0.0", Tools: []string{"search_issues"}},
			{Name: "github", Package: "x::other@9.9.9", Tools: []string{"run_command"}},
		},
	}

	updated := ServerEntry{Name: "GitHub", Package: "x::official@1.0.0", Tools: []string{"list_repos"}}
	require.NoError(t, cfg.UpdateServer(updated))

	require.Len(t, cfg.Servers, 2)
	require.Equal(t, updated, cfg.Servers[0])
	require.Equal(t, "github", cfg.Servers[1].Name)
	require.Equal(t, []string{"run_command"}, cfg.Servers[1].Tools)
}

// TestUpdateServer_MatchesWithoutRegardToCase pins that an update reaches the entry the configuration
// holds, whatever case the caller supplies.
func TestUpdateServer_MatchesWithoutRegardToCase(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		configFilePath: filepath.Join(t.TempDir(), ".mcpd.toml"),
		Servers:        []ServerEntry{{Name: "github", Package: "x::github@1.0.0"}},
	}

	require.NoError(t, cfg.UpdateServer(ServerEntry{Name: " GitHub ", Package: "x::github@2.0.0"}))

	require.Len(t, cfg.Servers, 1)
	require.Equal(t, "x::github@2.0.0", cfg.Servers[0].Package)
}

// TestUpdateServer_ErrorsIfNotFound pins that updating a server the configuration does not hold reports
// an error rather than adding it.
func TestUpdateServer_ErrorsIfNotFound(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		configFilePath: filepath.Join(t.TempDir(), ".mcpd.toml"),
		Servers:        []ServerEntry{{Name: "time", Package: "x::time@1.0.0"}},
	}

	err := cfg.UpdateServer(ServerEntry{Name: "github", Package: "x::github@1.0.0"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "server 'github' not found in config")
	require.Len(t, cfg.Servers, 1)
	require.Equal(t, "time", cfg.Servers[0].Name)
}

// TestUpdateServer_KeepsTheEntryWhenTheReplacementIsNotValid pins that a refused replacement leaves the
// configuration as it was.
func TestUpdateServer_KeepsTheEntryWhenTheReplacementIsNotValid(t *testing.T) {
	t.Parallel()

	original := ServerEntry{Name: "github", Package: "x::github@1.0.0"}
	cfg := &Config{
		configFilePath: filepath.Join(t.TempDir(), ".mcpd.toml"),
		Servers:        []ServerEntry{original},
	}

	require.Error(t, cfg.UpdateServer(ServerEntry{Name: "github", Package: "  "}))

	require.Len(t, cfg.Servers, 1)
	require.Equal(t, original, cfg.Servers[0])
}
