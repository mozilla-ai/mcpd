package config

import (
	"os"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/require"
)

func TestInit_CreatesNewConfigFile(t *testing.T) {
	t.Parallel()

	tempFile, err := os.CreateTemp(t.TempDir(), ".mcpd.toml")
	require.NoError(t, err)
	tempFilePath := tempFile.Name()
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
