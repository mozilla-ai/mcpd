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
			original := os.Getenv(EnvVarXDGConfigHome)
			t.Cleanup(func() {
				require.NoError(t, os.Setenv(EnvVarXDGConfigHome, original))
			})

			t.Setenv(EnvVarXDGConfigHome, tc.xdgValue)

			result, err := UserSpecificConfigDir()
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
