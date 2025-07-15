package context

import (
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
				require.Empty(t, cfg.ListServers())
			} else {
				require.Contains(t, cfg.ListServers(), "myserver")
				require.Equal(t, []string{"--foo", "--bar"}, cfg.ListServers()["myserver"].Args)
				require.Equal(t, "bar", cfg.ListServers()["myserver"].Env["FOO"])
			}
		})
	}
}

func TestSaveAndLoadExecutionContextConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Include extra, currently non-existing folder along the way.
	path := filepath.Join(dir, ".config", "mcpd", "secrets.dev.toml")

	original := &ExecutionContextConfig{
		Servers: map[string]ServerExecutionContext{
			"alpha": {
				Name: "alpha",
				Args: []string{"--debug"},
				Env:  map[string]string{"KEY": "VALUE"},
			},
		},
		filePath: path,
	}

	require.NoError(t, original.saveConfig())

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
