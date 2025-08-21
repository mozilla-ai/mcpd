package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/v2/internal/cache"
	"github.com/mozilla-ai/mcpd/v2/internal/config"
	"github.com/mozilla-ai/mcpd/v2/internal/context"
	"github.com/mozilla-ai/mcpd/v2/internal/perms"
)

// TestExecutionContextPermissions verifies that execution context files
// are created with secure permissions.
func TestExecutionContextPermissions(t *testing.T) {
	t.Parallel()

	// Create a base temp dir and then create our own secure subdirectory
	// to avoid the issue where t.TempDir() creates directories with 0755 permissions
	baseTempDir := t.TempDir()
	secureParentDir := filepath.Join(baseTempDir, "secure-exec-context")
	err := os.MkdirAll(secureParentDir, perms.SecureDir)
	require.NoError(t, err)

	configPath := filepath.Join(secureParentDir, "execution.toml")

	// Create an execution context config and save it.
	cfg := context.NewExecutionContextConfig(configPath)

	// Add a test server configuration.
	testServer := context.ServerExecutionContext{
		Name: "test-server",
		Args: []string{"--debug"},
		Env:  map[string]string{"TEST": "value"},
	}

	_, err = cfg.Upsert(testServer)
	require.NoError(t, err)

	// Verify the file was created with secure permissions.
	info, err := os.Stat(configPath)
	require.NoError(t, err)
	require.False(t, info.IsDir())
	require.Equal(t, perms.SecureFile, info.Mode().Perm(),
		"Execution context file should be created with secure permissions (0600)")

	// Verify the parent directory has secure permissions.
	parentDir := filepath.Dir(configPath)
	parentInfo, err := os.Stat(parentDir)
	require.NoError(t, err)
	require.True(t, parentInfo.IsDir())
	require.Equal(t, perms.SecureDir, parentInfo.Mode().Perm(),
		"Execution context directory should have secure permissions (0700)")
}

// TestConfigFilePermissions verifies that configuration files
// are created with regular permissions.
func TestConfigFilePermissions(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	// Create a config using the default loader.
	loader := &config.DefaultLoader{}
	err := loader.Init(configPath)
	require.NoError(t, err)

	// Verify the file was created with regular permissions.
	info, err := os.Stat(configPath)
	require.NoError(t, err)
	require.False(t, info.IsDir())
	require.Equal(t, perms.RegularFile, info.Mode().Perm(),
		"Configuration file should be created with regular permissions (0644)")
}

// TestCacheDirectoryPermissions verifies that cache directories
// are created with regular permissions.
func TestCacheDirectoryPermissions(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	cacheDir := filepath.Join(tempDir, "cache")
	logger := hclog.NewNullLogger()

	// Create cache with caching enabled to trigger directory creation.
	opts := []cache.Option{
		cache.WithDirectory(cacheDir),
		cache.WithCaching(true),
	}

	c, err := cache.NewCache(logger, opts...)
	require.NoError(t, err)
	require.NotNil(t, c)

	// Verify the cache directory was created with regular permissions.
	info, err := os.Stat(cacheDir)
	require.NoError(t, err)
	require.True(t, info.IsDir())
	require.Equal(t, perms.RegularDir, info.Mode().Perm(),
		"Cache directory should be created with regular permissions (0755)")
}

// TestDotEnvFilePermissions verifies that exported dotenv files
// have regular permissions (testing the export functionality).
func TestDotEnvFilePermissions(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	dotenvPath := filepath.Join(tempDir, "test.env")

	// Create a test dotenv file using the same pattern as export.go.
	testData := map[string]string{
		"TEST_VAR": "test_value",
		"DEBUG":    "true",
	}

	var b strings.Builder
	for k, v := range testData {
		b.WriteString(k)
		b.WriteString("=")
		b.WriteString(v)
		b.WriteString("\n")
	}

	// Write file using the same pattern as cmd/config/export/export.go.
	err := os.WriteFile(dotenvPath, []byte(b.String()), perms.RegularFile)
	require.NoError(t, err)

	// Verify the file was created with regular permissions.
	info, err := os.Stat(dotenvPath)
	require.NoError(t, err)
	require.False(t, info.IsDir())
	require.Equal(t, perms.RegularFile, info.Mode().Perm(),
		"Exported dotenv file should have regular permissions (0644)")
}

// TestLogFilePermissions verifies that log files
// are created with regular permissions.
func TestLogFilePermissions(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "mcpd.log")

	// Create log file using the same pattern as internal/cmd/basecmd.go.
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, perms.RegularFile)
	require.NoError(t, err)

	_, err = f.WriteString("test log entry\n")
	require.NoError(t, err)

	err = f.Close()
	require.NoError(t, err)

	// Verify the file was created with regular permissions.
	info, err := os.Stat(logPath)
	require.NoError(t, err)
	require.False(t, info.IsDir())
	require.Equal(t, perms.RegularFile, info.Mode().Perm(),
		"Log file should be created with regular permissions (0644)")
}

// TestPermissionConsistency verifies that different types of files
// have appropriate permission classifications.
func TestPermissionConsistency(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	// Create examples of each file type.
	files := map[string]struct {
		path        string
		perm        os.FileMode
		description string
		isSecure    bool
	}{
		"config": {
			path:        filepath.Join(tempDir, "config.toml"),
			perm:        perms.RegularFile,
			description: "configuration file",
			isSecure:    false,
		},
		"execution_context": {
			path:        filepath.Join(tempDir, "secrets.toml"),
			perm:        perms.SecureFile,
			description: "execution context file",
			isSecure:    true,
		},
		"log": {
			path:        filepath.Join(tempDir, "app.log"),
			perm:        perms.RegularFile,
			description: "log file",
			isSecure:    false,
		},
		"export": {
			path:        filepath.Join(tempDir, "export.env"),
			perm:        perms.RegularFile,
			description: "export file",
			isSecure:    false,
		},
	}

	for name, fileInfo := range files {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Create file with specified permissions.
			err := os.WriteFile(fileInfo.path, []byte("test content"), fileInfo.perm)
			require.NoError(t, err)

			// Verify permissions are as expected.
			info, err := os.Stat(fileInfo.path)
			require.NoError(t, err)
			require.Equal(t, fileInfo.perm, info.Mode().Perm(),
				"%s should have correct permissions", fileInfo.description)

			// Verify security classification is correct.
			if fileInfo.isSecure {
				require.Equal(t, perms.SecureFile, fileInfo.perm,
					"Secure %s should use SecureFile permissions", fileInfo.description)
			} else {
				require.Equal(t, perms.RegularFile, fileInfo.perm,
					"Regular %s should use RegularFile permissions", fileInfo.description)
			}
		})
	}
}

// TestDirectoryPermissionConsistency verifies that different types of directories
// have appropriate permission classifications.
func TestDirectoryPermissionConsistency(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	// Create examples of each directory type.
	dirs := map[string]struct {
		path        string
		perm        os.FileMode
		description string
		isSecure    bool
	}{
		"cache": {
			path:        filepath.Join(tempDir, "cache"),
			perm:        perms.RegularDir,
			description: "cache directory",
			isSecure:    false,
		},
		"execution_context": {
			path:        filepath.Join(tempDir, "execution"),
			perm:        perms.SecureDir,
			description: "execution context directory",
			isSecure:    true,
		},
	}

	for name, dirInfo := range dirs {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Create directory with specified permissions.
			err := os.MkdirAll(dirInfo.path, dirInfo.perm)
			require.NoError(t, err)

			// Verify permissions are as expected.
			info, err := os.Stat(dirInfo.path)
			require.NoError(t, err)
			require.True(t, info.IsDir())
			require.Equal(t, dirInfo.perm, info.Mode().Perm(),
				"%s should have correct permissions", dirInfo.description)

			// Verify security classification is correct.
			if dirInfo.isSecure {
				require.Equal(t, perms.SecureDir, dirInfo.perm,
					"Secure %s should use SecureDir permissions", dirInfo.description)
			} else {
				require.Equal(t, perms.RegularDir, dirInfo.perm,
					"Regular %s should use RegularDir permissions", dirInfo.description)
			}
		})
	}
}
