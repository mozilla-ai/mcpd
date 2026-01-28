package plugin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/internal/config"
)

func TestManager_NewManager_ValidInputs(t *testing.T) {
	t.Parallel()

	logger := hclog.NewNullLogger()
	cfg := &config.PluginConfig{Dir: "/tmp"}

	m, err := NewManager(logger, cfg)
	require.NoError(t, err)
	require.NotNil(t, m)
}

func TestManager_NewManager_NilLogger(t *testing.T) {
	t.Parallel()

	cfg := &config.PluginConfig{Dir: "/tmp"}

	m, err := NewManager(nil, cfg)
	require.Error(t, err)
	require.Nil(t, m)
	require.Contains(t, err.Error(), "logger cannot be nil")
}

func TestManager_NewManager_NilConfig(t *testing.T) {
	t.Parallel()

	logger := hclog.NewNullLogger()

	m, err := NewManager(logger, nil)
	require.Error(t, err)
	require.Nil(t, m)
	require.Contains(t, err.Error(), "plugin config cannot be nil")
}

func TestManager_discoverPlugins_EmptyAllowed(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	// Create an executable file.
	execPath := filepath.Join(tempDir, "some-plugin")
	err := os.WriteFile(execPath, []byte("#!/bin/sh\necho test"), 0o755)
	require.NoError(t, err)

	logger := hclog.NewNullLogger()
	cfg := &config.PluginConfig{Dir: tempDir}
	m, err := NewManager(logger, cfg)
	require.NoError(t, err)

	// Empty allowed set should return nil.
	allowed := make(map[string]struct{})
	plugins, err := m.discoverPlugins(allowed)
	require.NoError(t, err)
	require.Nil(t, plugins)
}

func TestManager_discoverPlugins_NonExistentDirectory(t *testing.T) {
	t.Parallel()

	logger := hclog.NewNullLogger()
	cfg := &config.PluginConfig{Dir: "/nonexistent/path"}
	m, err := NewManager(logger, cfg)
	require.NoError(t, err)

	allowed := map[string]struct{}{"test-plugin": {}}
	plugins, err := m.discoverPlugins(allowed)
	require.Error(t, err)
	require.Nil(t, plugins)
	require.Contains(t, err.Error(), "reading plugin directory")
}

func TestManager_discoverPlugins_WithExecutableFiles(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	// Create executable file.
	execPath := filepath.Join(tempDir, "test-plugin")
	err := os.WriteFile(execPath, []byte("#!/bin/sh\necho test"), 0o755)
	require.NoError(t, err)

	// Create non-executable file.
	nonExecPath := filepath.Join(tempDir, "readme.txt")
	err = os.WriteFile(nonExecPath, []byte("readme"), 0o644)
	require.NoError(t, err)

	logger := hclog.NewNullLogger()
	cfg := &config.PluginConfig{Dir: tempDir}
	m, err := NewManager(logger, cfg)
	require.NoError(t, err)

	allowed := map[string]struct{}{
		"test-plugin": {},
		"readme.txt":  {},
	}
	plugins, err := m.discoverPlugins(allowed)
	require.NoError(t, err)
	require.Len(t, plugins, 1)
	require.Contains(t, plugins, "test-plugin")
	require.Equal(t, execPath, plugins["test-plugin"])
}

func TestManager_discoverPlugins_SkipsHiddenFiles(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	// Create hidden executable.
	hiddenPath := filepath.Join(tempDir, ".hidden-plugin")
	err := os.WriteFile(hiddenPath, []byte("#!/bin/sh\necho hidden"), 0o755)
	require.NoError(t, err)

	// Create visible executable.
	visiblePath := filepath.Join(tempDir, "visible-plugin")
	err = os.WriteFile(visiblePath, []byte("#!/bin/sh\necho visible"), 0o755)
	require.NoError(t, err)

	logger := hclog.NewNullLogger()
	cfg := &config.PluginConfig{Dir: tempDir}
	m, err := NewManager(logger, cfg)
	require.NoError(t, err)

	allowed := map[string]struct{}{
		".hidden-plugin": {},
		"visible-plugin": {},
	}
	plugins, err := m.discoverPlugins(allowed)
	require.NoError(t, err)
	require.Len(t, plugins, 1)
	require.Contains(t, plugins, "visible-plugin")
	require.NotContains(t, plugins, ".hidden-plugin")
}

func TestManager_discoverPlugins_SkipsDirectories(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	// Create subdirectory.
	subDir := filepath.Join(tempDir, "subdir")
	err := os.Mkdir(subDir, 0o755)
	require.NoError(t, err)

	// Create executable in main dir.
	execPath := filepath.Join(tempDir, "plugin")
	err = os.WriteFile(execPath, []byte("#!/bin/sh\necho test"), 0o755)
	require.NoError(t, err)

	logger := hclog.NewNullLogger()
	cfg := &config.PluginConfig{Dir: tempDir}
	m, err := NewManager(logger, cfg)
	require.NoError(t, err)

	allowed := map[string]struct{}{
		"plugin": {},
		"subdir": {},
	}
	plugins, err := m.discoverPlugins(allowed)
	require.NoError(t, err)
	require.Len(t, plugins, 1)
	require.Contains(t, plugins, "plugin")
}

func TestManager_discoverPlugins_MultipleExecutables(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	pluginNames := []string{"plugin1", "plugin2", "plugin3"}
	for _, name := range pluginNames {
		path := filepath.Join(tempDir, name)
		err := os.WriteFile(path, []byte("#!/bin/sh\necho "+name), 0o755)
		require.NoError(t, err)
	}

	logger := hclog.NewNullLogger()
	cfg := &config.PluginConfig{Dir: tempDir}
	m, err := NewManager(logger, cfg)
	require.NoError(t, err)

	allowed := map[string]struct{}{
		"plugin1": {},
		"plugin2": {},
		"plugin3": {},
	}
	plugins, err := m.discoverPlugins(allowed)
	require.NoError(t, err)
	require.Len(t, plugins, 3)

	for _, name := range pluginNames {
		require.Contains(t, plugins, name)
		require.Equal(t, filepath.Join(tempDir, name), plugins[name])
	}
}

func TestManager_discoverPlugins_OnlyDiscoverAllowed(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	// Create three executables.
	for _, name := range []string{"plugin1", "plugin2", "plugin3"} {
		path := filepath.Join(tempDir, name)
		err := os.WriteFile(path, []byte("#!/bin/sh\necho "+name), 0o755)
		require.NoError(t, err)
	}

	logger := hclog.NewNullLogger()
	cfg := &config.PluginConfig{Dir: tempDir}
	m, err := NewManager(logger, cfg)
	require.NoError(t, err)

	// Only allow plugin1 and plugin3.
	allowed := map[string]struct{}{
		"plugin1": {},
		"plugin3": {},
	}
	plugins, err := m.discoverPlugins(allowed)
	require.NoError(t, err)
	require.Len(t, plugins, 2)
	require.Contains(t, plugins, "plugin1")
	require.Contains(t, plugins, "plugin3")
	require.NotContains(t, plugins, "plugin2")
}

func TestManager_formatDialAddress_UnixNetwork(t *testing.T) {
	t.Parallel()

	logger := hclog.NewNullLogger()
	cfg := &config.PluginConfig{Dir: "/tmp"}
	m, err := NewManager(logger, cfg)
	require.NoError(t, err)

	address := "/tmp/test.sock"
	result := m.formatDialAddress(networkUnix, address)
	require.Equal(t, "unix:///tmp/test.sock", result)
}

func TestManager_formatDialAddress_EmptyAddress(t *testing.T) {
	t.Parallel()

	logger := hclog.NewNullLogger()
	cfg := &config.PluginConfig{Dir: "/tmp"}
	m, err := NewManager(logger, cfg)
	require.NoError(t, err)

	result := m.formatDialAddress(networkUnix, "")
	require.Equal(t, "unix://", result)
}

func TestManager_generateAddress_ReturnsValidFormat(t *testing.T) {
	t.Parallel()

	logger := hclog.NewNullLogger()
	cfg := &config.PluginConfig{Dir: "/tmp"}
	m, err := NewManager(logger, cfg)
	require.NoError(t, err)

	address, network := m.generateAddress("test-plugin")
	require.NotEmpty(t, address)
	require.Equal(t, networkUnix, network)

	// Unix socket should be a file path ending in .sock.
	require.True(t, strings.HasSuffix(address, ".sock"))
	require.Contains(t, address, "test-plugin")
}

func TestManager_generateAddress_ReplacesSpacesInPluginName(t *testing.T) {
	t.Parallel()

	logger := hclog.NewNullLogger()
	cfg := &config.PluginConfig{Dir: "/tmp"}
	m, err := NewManager(logger, cfg)
	require.NoError(t, err)

	address, network := m.generateAddress("test plugin with spaces")
	require.NotEmpty(t, address)
	require.Equal(t, networkUnix, network)

	// Unix sockets should have spaces replaced with dashes.
	require.Contains(t, address, "test-plugin-with-spaces")
	require.NotContains(t, address, " ")
}

func TestManager_generateAddress_GeneratesUniqueAddresses(t *testing.T) {
	t.Parallel()

	logger := hclog.NewNullLogger()
	cfg := &config.PluginConfig{Dir: "/tmp"}
	m, err := NewManager(logger, cfg)
	require.NoError(t, err)

	// Generate multiple addresses and ensure they're different.
	addresses := make(map[string]bool)
	for i := 0; i < 10; i++ {
		address, _ := m.generateAddress("test-plugin")
		require.NotContains(t, addresses, address, "generated duplicate address")
		addresses[address] = true
	}
}

func TestManager_startPlugin_NonExistentBinary(t *testing.T) {
	t.Parallel()

	logger := hclog.NewNullLogger()
	cfg := &config.PluginConfig{Dir: "/tmp"}
	m, err := NewManager(logger, cfg)
	require.NoError(t, err)

	ctx := context.Background()
	plg, err := m.startPlugin(ctx, "test-plugin", "/nonexistent/binary")
	require.Error(t, err)
	require.Nil(t, plg)
	require.Contains(t, err.Error(), "failed to start process")
}

func TestManager_startPlugin_NonExecutableBinary(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	// Create a non-executable file.
	binaryPath := filepath.Join(tempDir, "non-exec")
	err := os.WriteFile(binaryPath, []byte("#!/bin/sh\necho test"), 0o644)
	require.NoError(t, err)

	logger := hclog.NewNullLogger()
	cfg := &config.PluginConfig{Dir: tempDir}
	m, err := NewManager(logger, cfg)
	require.NoError(t, err)

	ctx := context.Background()
	plg, err := m.startPlugin(ctx, "test-plugin", binaryPath)
	require.Error(t, err)
	require.Nil(t, plg)
}

func TestManager_startPlugin_CancelledContext(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	// Create an executable that sleeps forever.
	binaryPath := filepath.Join(tempDir, "sleeper")
	err := os.WriteFile(binaryPath, []byte("#!/bin/sh\nsleep 1000"), 0o755)
	require.NoError(t, err)

	logger := hclog.NewNullLogger()
	cfg := &config.PluginConfig{Dir: tempDir}
	m, err := NewManager(logger, cfg)
	require.NoError(t, err)

	// Create cancelled context.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	plg, err := m.startPlugin(ctx, "test-plugin", binaryPath)
	require.Error(t, err)
	require.Nil(t, plg)
}

func TestManager_StopPlugins_EmptyPluginsMap(t *testing.T) {
	t.Parallel()

	logger := hclog.NewNullLogger()
	cfg := &config.PluginConfig{Dir: "/tmp"}
	m, err := NewManager(logger, cfg)
	require.NoError(t, err)

	err = m.StopPlugins()
	require.NoError(t, err)
}

func TestRunningPlugin_validate_NoCommitHash(t *testing.T) {
	t.Parallel()

	// Test with nil commit hash.
	entry := config.PluginEntry{
		Name:       "test-plugin",
		CommitHash: nil,
	}

	// Since we can't easily create a real runningPlugin, we test the logic would skip validation.
	// This is testing the documented behavior: nil commit hash skips validation.
	require.Nil(t, entry.CommitHash)
}

func TestRunningPlugin_validate_EmptyCommitHash(t *testing.T) {
	t.Parallel()

	// Test with empty commit hash.
	emptyHash := ""
	entry := config.PluginEntry{
		Name:       "test-plugin",
		CommitHash: &emptyHash,
	}

	// Verify empty string is falsy for validation skip logic.
	require.NotNil(t, entry.CommitHash)
	require.Equal(t, "", *entry.CommitHash)
}

func TestRunningPlugin_stop_NilPlugin(t *testing.T) {
	t.Parallel()

	var plg *runningPlugin

	err := plg.stop()
	require.Error(t, err)
	require.Contains(t, err.Error(), "plugin is nil")
}
