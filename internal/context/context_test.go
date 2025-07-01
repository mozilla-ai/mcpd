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
			cfg, err := LoadOrInitExecutionContext(path)

			if tc.wantErr {
				require.Error(t, err)
				require.ErrorContains(t, err, "failed to load execution context config")
				require.ErrorContains(t, err, "could not be parsed")
				return
			}

			require.NoError(t, err)
			if tc.expectInit {
				require.Empty(t, cfg.Servers)
			} else {
				require.Contains(t, cfg.Servers, "myserver")
				require.Equal(t, []string{"--foo", "--bar"}, cfg.Servers["myserver"].Args)
				require.Equal(t, "bar", cfg.Servers["myserver"].Env["FOO"])
			}
		})
	}
}

func TestSaveAndLoadExecutionContextConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.dev.toml")

	original := ExecutionContextConfig{
		Servers: map[string]ServerExecutionContext{
			"alpha": {
				Args: []string{"--debug"},
				Env:  map[string]string{"KEY": "VALUE"},
			},
		},
	}

	require.NoError(t, SaveExecutionContextConfig(path, original))
	loaded, err := LoadExecutionContextConfig(path)
	require.NoError(t, err)
	require.Equal(t, original, loaded)
}
