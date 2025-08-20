package context

import (
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadOrInitExecutionContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setup      func(t *testing.T) string
		wantErr    bool
		expectInit bool
	}{
		{
			name: "file does not exist - returns initialized config",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent.toml")
			},
			wantErr:    false,
			expectInit: true,
		},
		{
			name: "file exists and is valid",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "valid.toml")
				content := `
[servers.myserver]
args = ["--foo", "--bar"]
[servers.myserver.env]
FOO = "bar"
`
				err := os.WriteFile(path, []byte(content), 0o644)
				require.NoError(t, err)
				return path
			},
			wantErr:    false,
			expectInit: false,
		},
		{
			name: "file exists but is malformed",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "bad.toml")
				content := "not [valid"
				err := os.WriteFile(path, []byte(content), 0o644)
				require.NoError(t, err)
				return path
			},
			wantErr:    true,
			expectInit: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			path := tc.setup(t)
			loader := DefaultLoader{}
			cfg, err := loader.Load(path)

			if tc.wantErr {
				require.Error(t, err)
				require.ErrorContains(t, err, "failed to load execution context config")
				require.ErrorContains(t, err, "could not be parsed")
				return
			}

			require.NoError(t, err)

			if tc.expectInit {
				_, ok := cfg.Get("myserver")
				require.False(t, ok)
			} else {
				server, ok := cfg.Get("myserver")
				require.True(t, ok)
				require.Equal(t, []string{"--foo", "--bar"}, server.Args)
				require.Equal(t, "bar", server.Env["FOO"])
			}
		})
	}
}

func TestSaveAndLoadExecutionContextConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Include extra, currently non-existing folder along the way.
	path := filepath.Join(dir, ".config", "mcpd", "secrets.dev.toml")

	original := NewExecutionContextConfig(path)
	original.Servers = map[string]ServerExecutionContext{
		"alpha": {
			Name: "alpha",
			Args: []string{"--debug"},
			Env:  map[string]string{"KEY": "VALUE"},
		},
	}

	require.NoError(t, original.SaveConfig())

	loader := DefaultLoader{}
	loaded, err := loader.Load(path)
	require.NoError(t, err)

	require.Equal(t, original, loaded)
}

func TestAppDirName(t *testing.T) {
	t.Parallel()

	require.Equal(t, "mcpd", AppDirName())
}

func TestContext_UserSpecificConfigDir(t *testing.T) {
	tests := []struct {
		name        string
		xdgValue    string
		expectedDir func(t *testing.T) string
	}{
		{
			name:     "XDG_CONFIG_HOME is set and used",
			xdgValue: "/custom/xdg/path",
			expectedDir: func(t *testing.T) string {
				return filepath.Join("/custom/xdg/path", AppDirName())
			},
		},
		{
			name:     "XDG_CONFIG_HOME is set with whitespace and trimmed",
			xdgValue: "  /trimmed/xdg/path  ",
			expectedDir: func(t *testing.T) string {
				return filepath.Join("/trimmed/xdg/path", AppDirName())
			},
		},
		{
			name:     "XDG_CONFIG_HOME is empty, fall back to default",
			xdgValue: "",
			expectedDir: func(t *testing.T) string {
				home, err := os.UserHomeDir()
				require.NoError(t, err)
				return filepath.Join(home, ".config", AppDirName())
			},
		},
		{
			name:     "XDG_CONFIG_HOME is only whitespace, fall back to default",
			xdgValue: "   ",
			expectedDir: func(t *testing.T) string {
				home, err := os.UserHomeDir()
				require.NoError(t, err)
				return filepath.Join(home, ".config", AppDirName())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(EnvVarXDGConfigHome, tc.xdgValue)

			result, err := UserSpecificConfigDir()
			require.NoError(t, err)
			require.Equal(t, tc.expectedDir(t), result)
		})
	}
}

func TestUserSpecificDir_InvalidEnvVar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		envVar string
		dir    string
	}{
		{
			name:   "environment variable without XDG_ prefix",
			envVar: "CONFIG_HOME",
			dir:    ".config",
		},
		{
			name:   "empty environment variable name",
			envVar: "",
			dir:    ".cache",
		},
		{
			name:   "environment variable with wrong prefix",
			envVar: "CACHE_HOME",
			dir:    ".cache",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := userSpecificDir(tc.envVar, tc.dir)
			require.Error(t, err)
			require.Contains(t, err.Error(), "does not follow XDG Base Directory Specification")
		})
	}
}

func TestContext_UserSpecificCacheDir(t *testing.T) {
	tests := []struct {
		name        string
		xdgValue    string
		expectedDir func(t *testing.T) string
	}{
		{
			name:     "XDG_CACHE_HOME is set and used",
			xdgValue: "/custom/cache/path",
			expectedDir: func(t *testing.T) string {
				return filepath.Join("/custom/cache/path", AppDirName())
			},
		},
		{
			name:     "XDG_CACHE_HOME is set with whitespace and trimmed",
			xdgValue: "  /trimmed/cache/path  ",
			expectedDir: func(t *testing.T) string {
				return filepath.Join("/trimmed/cache/path", AppDirName())
			},
		},
		{
			name:     "XDG_CACHE_HOME is empty, fall back to default",
			xdgValue: "",
			expectedDir: func(t *testing.T) string {
				home, err := os.UserHomeDir()
				require.NoError(t, err)
				return filepath.Join(home, ".cache", AppDirName())
			},
		},
		{
			name:     "XDG_CACHE_HOME is only whitespace, fall back to default",
			xdgValue: "   ",
			expectedDir: func(t *testing.T) string {
				home, err := os.UserHomeDir()
				require.NoError(t, err)
				return filepath.Join(home, ".cache", AppDirName())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(EnvVarXDGCacheHome, tc.xdgValue)

			result, err := UserSpecificCacheDir()
			require.NoError(t, err)
			require.Equal(t, tc.expectedDir(t), result)
		})
	}
}

func TestUpsert(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		existing       map[string]ServerExecutionContext
		input          ServerExecutionContext
		expectedResult UpsertResult
		verify         func(t *testing.T, cfg *ExecutionContextConfig, path string)
	}{
		{
			name:     "new empty",
			existing: nil,
			input: ServerExecutionContext{
				Name: "foo",
				Args: []string{},
				Env:  map[string]string{},
			},
			expectedResult: Noop,
			verify: func(t *testing.T, cfg *ExecutionContextConfig, path string) {
				_, exists := cfg.Servers["foo"]
				require.False(t, exists)
				_, err := os.Stat(path)
				require.Error(t, err)
			},
		},
		{
			name:     "new non empty",
			existing: nil,
			input: ServerExecutionContext{
				Name: "bar",
				Args: []string{"--foo"},
				Env:  map[string]string{"KEY": "VAL"},
			},
			expectedResult: Created,
			verify: func(t *testing.T, cfg *ExecutionContextConfig, path string) {
				got, exists := cfg.Servers["bar"]
				require.True(t, exists)
				require.Equal(t, []string{"--foo"}, got.Args)
				require.Equal(t, map[string]string{"KEY": "VAL"}, got.Env)
				fi, err := os.Stat(path)
				require.NoError(t, err)
				require.Greater(t, fi.Size(), int64(0))
			},
		},
		{
			name: "existing same",
			existing: map[string]ServerExecutionContext{
				"baz": {
					Name: "baz",
					Args: []string{"--bar"},
					Env:  map[string]string{"DEBUG": "1"},
				},
			},
			input: ServerExecutionContext{
				Name: "baz",
				Args: []string{"--bar"},
				Env:  map[string]string{"DEBUG": "1"},
			},
			expectedResult: Noop,
			verify: func(t *testing.T, cfg *ExecutionContextConfig, path string) {
				// File shouldn't exist since we would never have tried to write one.
				_, err := os.Stat(path)
				require.Error(t, err)
			},
		},
		{
			name: "existing updated",
			existing: map[string]ServerExecutionContext{
				"baz": {
					Name: "baz",
					Args: []string{"--bar"},
					Env:  map[string]string{"DEBUG": "1"},
				},
			},
			input: ServerExecutionContext{
				Name: "baz",
				Args: []string{"--bar", "--extra"},
				Env:  map[string]string{"DEBUG": "1"},
			},
			expectedResult: Updated,
			verify: func(t *testing.T, cfg *ExecutionContextConfig, path string) {
				got := cfg.Servers["baz"]
				require.Equal(t, []string{"--bar", "--extra"}, got.Args)
				fi, err := os.Stat(path)
				require.NoError(t, err)
				require.Greater(t, fi.Size(), int64(0))
			},
		},
		{
			name: "existing cleared",
			existing: map[string]ServerExecutionContext{
				"baz": {
					Name: "baz",
					Args: []string{"--bar"},
					Env:  map[string]string{"DEBUG": "1"},
				},
			},
			input: ServerExecutionContext{
				Name: "baz",
				Args: []string{},
				Env:  map[string]string{},
			},
			expectedResult: Deleted,
			verify: func(t *testing.T, cfg *ExecutionContextConfig, path string) {
				_, exists := cfg.Servers["baz"]
				require.False(t, exists)
				fi, err := os.Stat(path)
				require.NoError(t, err)
				require.Greater(t, fi.Size(), int64(0))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			path := filepath.Join(dir, ".config", "mcpd", "secrets.test.toml")

			cfg := &ExecutionContextConfig{
				Servers:  maps.Clone(tc.existing),
				filePath: path,
			}

			result, err := cfg.Upsert(tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.expectedResult, result)
			tc.verify(t, cfg, path)
		})
	}
}

// TestLoadExecutionContextConfig_VariableExpansion tests that ${VAR} references are expanded at load time.
func TestLoadExecutionContextConfig_VariableExpansion(t *testing.T) {
	// Note: Cannot use t.Parallel() because subtests use t.Setenv

	testCases := []struct {
		name     string
		content  string
		envVars  map[string]string
		expected map[string]ServerExecutionContext
	}{
		{
			name: "expand environment variables in env section",
			content: `[servers.test-server]
args = ["--port", "8080"]
[servers.test-server.env]
API_KEY = "${TEST_API_KEY}"
DB_URL = "${TEST_DB_URL}"
LITERAL = "no-expansion"`,
			envVars: map[string]string{
				"TEST_API_KEY": "secret-key-123",
				"TEST_DB_URL":  "postgres://localhost:5432/test",
			},
			expected: map[string]ServerExecutionContext{
				"test-server": {
					Name: "test-server",
					Args: []string{"--port", "8080"},
					Env: map[string]string{
						"API_KEY": "secret-key-123",
						"DB_URL":  "postgres://localhost:5432/test",
						"LITERAL": "no-expansion",
					},
				},
			},
		},
		{
			name: "expand environment variables in args section",
			content: `[servers.test-server]
args = ["--token", "${AUTH_TOKEN}", "--config", "${CONFIG_PATH}", "--literal", "unchanged"]
[servers.test-server.env]
DEBUG = "true"`,
			envVars: map[string]string{
				"AUTH_TOKEN":  "bearer-token-456",
				"CONFIG_PATH": "/etc/myapp/config.json",
			},
			expected: map[string]ServerExecutionContext{
				"test-server": {
					Name: "test-server",
					Args: []string{
						"--token",
						"bearer-token-456",
						"--config",
						"/etc/myapp/config.json",
						"--literal",
						"unchanged",
					},
					Env: map[string]string{
						"DEBUG": "true",
					},
				},
			},
		},
		{
			name: "expand variables with KEY=VALUE format in args",
			content: `[servers.test-server]
args = ["--api-key=${API_KEY}", "CONFIG_FILE=${CONFIG_FILE}", "--standalone", "${STANDALONE_VAR}"]`,
			envVars: map[string]string{
				"API_KEY":        "key-789",
				"CONFIG_FILE":    "/path/to/config",
				"STANDALONE_VAR": "standalone-value",
			},
			expected: map[string]ServerExecutionContext{
				"test-server": {
					Name: "test-server",
					Args: []string{
						"--api-key=key-789",
						"CONFIG_FILE=/path/to/config",
						"--standalone",
						"standalone-value",
					},
					Env: nil, // No env section in TOML means nil map
				},
			},
		},
		{
			name: "non-existent variables expand to empty string",
			content: `[servers.test-server]
args = ["--missing", "${NON_EXISTENT_VAR}", "--empty=${ANOTHER_MISSING}"]
[servers.test-server.env]
MISSING_VALUE = "${UNDEFINED_ENV_VAR}"
PRESENT_VALUE = "${DEFINED_VAR}"`,
			envVars: map[string]string{
				"DEFINED_VAR": "defined-value",
			},
			expected: map[string]ServerExecutionContext{
				"test-server": {
					Name: "test-server",
					Args: []string{"--missing", "", "--empty="},
					Env: map[string]string{
						"MISSING_VALUE": "",
						"PRESENT_VALUE": "defined-value",
					},
				},
			},
		},
		{
			name: "multiple servers with different expansions",
			content: `[servers.server-a]
args = ["--port", "${SERVER_A_PORT}"]
[servers.server-a.env]
SERVICE_NAME = "${SERVER_A_NAME}"

[servers.server-b]
args = ["--port", "${SERVER_B_PORT}"]
[servers.server-b.env]
SERVICE_NAME = "${SERVER_B_NAME}"`,
			envVars: map[string]string{
				"SERVER_A_PORT": "3000",
				"SERVER_A_NAME": "service-alpha",
				"SERVER_B_PORT": "4000",
				"SERVER_B_NAME": "service-beta",
			},
			expected: map[string]ServerExecutionContext{
				"server-a": {
					Name: "server-a",
					Args: []string{"--port", "3000"},
					Env: map[string]string{
						"SERVICE_NAME": "service-alpha",
					},
				},
				"server-b": {
					Name: "server-b",
					Args: []string{"--port", "4000"},
					Env: map[string]string{
						"SERVICE_NAME": "service-beta",
					},
				},
			},
		},
		{
			name: "complex variable references",
			content: `[servers.complex-server]
args = ["--url", "${PROTO}://${HOST}:${PORT}${PATH}", "--config", "${HOME}/.config/app.json"]
[servers.complex-server.env]
FULL_URL = "${PROTO}://${HOST}:${PORT}${PATH}"
HOME_CONFIG = "${HOME}/.config"`,
			envVars: map[string]string{
				"PROTO": "https",
				"HOST":  "api.example.com",
				"PORT":  "443",
				"PATH":  "/v1/api",
				"HOME":  "/home/user",
			},
			expected: map[string]ServerExecutionContext{
				"complex-server": {
					Name: "complex-server",
					Args: []string{
						"--url",
						"https://api.example.com:443/v1/api",
						"--config",
						"/home/user/.config/app.json",
					},
					Env: map[string]string{
						"FULL_URL":    "https://api.example.com:443/v1/api",
						"HOME_CONFIG": "/home/user/.config",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Note: Cannot use t.Parallel() with t.Setenv

			// Set up environment variables.
			for k, v := range tc.envVars {
				t.Setenv(k, v)
			}

			// Create a temporary config file.
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "test-config.toml")
			err := os.WriteFile(configPath, []byte(tc.content), 0o644)
			require.NoError(t, err)

			// Load the config.
			cfg, err := loadExecutionContextConfig(configPath)
			require.NoError(t, err)
			require.NotNil(t, cfg)

			// Verify the expanded results.
			require.Equal(t, len(tc.expected), len(cfg.Servers), "Should have expected number of servers")

			for expectedName, expectedServer := range tc.expected {
				actualServer, exists := cfg.Servers[expectedName]
				require.True(t, exists, "Server %s should exist", expectedName)

				// Check server name.
				require.Equal(t, expectedServer.Name, actualServer.Name, "Server name should match")

				// Check args.
				require.Equal(
					t,
					expectedServer.Args,
					actualServer.Args,
					"Args should be expanded correctly for server %s",
					expectedName,
				)

				// Check env vars.
				require.Equal(
					t,
					expectedServer.Env,
					actualServer.Env,
					"Env vars should be expanded correctly for server %s",
					expectedName,
				)
			}
		})
	}
}

// TestLoadExecutionContextConfig_NoExpansionWhenNoVars tests that files without variables work normally.
func TestLoadExecutionContextConfig_NoExpansionWhenNoVars(t *testing.T) {
	t.Parallel()

	content := `[servers.simple-server]
args = ["--port", "3000", "--debug"]
[servers.simple-server.env]
NODE_ENV = "development"
DEBUG = "true"`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "simple-config.toml")
	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	cfg, err := loadExecutionContextConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	expected := map[string]ServerExecutionContext{
		"simple-server": {
			Name: "simple-server",
			Args: []string{"--port", "3000", "--debug"},
			Env: map[string]string{
				"NODE_ENV": "development",
				"DEBUG":    "true",
			},
		},
	}

	require.Equal(t, expected, cfg.Servers)
}

// TestLoadExecutionContextConfig_UndefinedVariables tests that undefined environment variables expand to empty strings.
func TestLoadExecutionContextConfig_UndefinedVariables(t *testing.T) {
	t.Parallel()

	content := `[servers.test-server]
args = ["--token", "${UNDEFINED_TOKEN}", "--config=${UNDEFINED_CONFIG_PATH}"]
[servers.test-server.env]
API_KEY = "${UNDEFINED_API_KEY}"
DATABASE_URL = "${UNDEFINED_DB_URL}"`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "undefined-vars-config.toml")
	err := os.WriteFile(configPath, []byte(content), 0o644)
	require.NoError(t, err)

	// Deliberately NOT setting any environment variables
	cfg, err := loadExecutionContextConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	server, exists := cfg.Servers["test-server"]
	require.True(t, exists, "test-server should exist")

	// All undefined variables should expand to empty strings
	require.Equal(
		t,
		[]string{"--token", "", "--config="},
		server.Args,
		"Undefined vars in args should expand to empty strings",
	)
	require.Equal(t, map[string]string{
		"API_KEY":      "",
		"DATABASE_URL": "",
	}, server.Env, "Undefined vars in env should expand to empty strings")
}

// TestLoadExecutionContextConfig_EmptyFile tests loading an empty config file.
func TestLoadExecutionContextConfig_EmptyFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "empty-config.toml")
	err := os.WriteFile(configPath, []byte(""), 0o644)
	require.NoError(t, err)

	cfg, err := loadExecutionContextConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Empty(t, cfg.Servers, "Empty file should result in no servers")
}
