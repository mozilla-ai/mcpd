package files

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/internal/perms"
)

func TestAppDirName(t *testing.T) {
	t.Parallel()

	require.Equal(t, "mcpd", AppDirName())
}

func TestUserSpecificConfigDir(t *testing.T) {
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

func TestUserSpecificCacheDir(t *testing.T) {
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
			require.ErrorContains(t, err, "does not follow XDG Base Directory Specification")
		})
	}
}

func TestIsPermissionAcceptable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		actual   os.FileMode
		required os.FileMode
		want     bool
	}{
		// Exact matches should always be acceptable.
		{
			name:     "exact match 0755",
			actual:   0o755,
			required: 0o755,
			want:     true,
		},
		{
			name:     "exact match 0700",
			actual:   0o700,
			required: 0o700,
			want:     true,
		},
		{
			name:     "exact match 0644",
			actual:   0o644,
			required: 0o644,
			want:     true,
		},
		// More restrictive should be acceptable.
		{
			name:     "0700 is acceptable when 0755 is required",
			actual:   0o700,
			required: 0o755,
			want:     true,
		},
		{
			name:     "0600 is acceptable when 0644 is required",
			actual:   0o600,
			required: 0o644,
			want:     true,
		},
		{
			name:     "0000 is acceptable for any requirement (most restrictive)",
			actual:   0o000,
			required: 0o755,
			want:     true,
		},
		// Less restrictive should NOT be acceptable.
		{
			name:     "0755 is not acceptable when 0700 is required",
			actual:   0o755,
			required: 0o700,
			want:     false,
		},
		{
			name:     "0777 is not acceptable when 0755 is required",
			actual:   0o777,
			required: 0o755,
			want:     false,
		},
		{
			name:     "0666 is not acceptable when 0644 is required",
			actual:   0o666,
			required: 0o644,
			want:     false,
		},
		// Different permission patterns.
		{
			name:     "0711 is acceptable when 0755 is required (more restrictive for group/others)",
			actual:   0o711,
			required: 0o755,
			want:     true,
		},
		{
			name:     "0750 is acceptable when 0755 is required (more restrictive for others)",
			actual:   0o750,
			required: 0o755,
			want:     true,
		},
		{
			name:     "0705 is acceptable when 0755 is required (more restrictive for group)",
			actual:   0o705,
			required: 0o755,
			want:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isPermissionAcceptable(tc.actual, tc.required)
			require.Equal(
				t,
				tc.want,
				got,
				"isPermissionAcceptable(%#o, %#o) should return %v",
				tc.actual,
				tc.required,
				tc.want,
			)
		})
	}
}

func TestEnsureAtLeastSecureDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		wantErr bool
		errMsg  string
	}{
		{
			name: "creates directory when it doesn't exist",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "new-secure-dir")
			},
			wantErr: false,
		},
		{
			name: "accepts directory with exact required permissions",
			setup: func(t *testing.T) string {
				dir := filepath.Join(t.TempDir(), "exact-perms")
				require.NoError(t, os.Mkdir(dir, perms.SecureDir))
				return dir
			},
			wantErr: false,
		},
		{
			name: "accepts directory with 0o600 (more restrictive)",
			setup: func(t *testing.T) string {
				dir := filepath.Join(t.TempDir(), "more-restrictive")
				require.NoError(t, os.Mkdir(dir, 0o600))
				return dir
			},
			wantErr: false,
		},
		{
			name: "rejects directory with less restrictive permissions",
			setup: func(t *testing.T) string {
				dir := filepath.Join(t.TempDir(), "less-restrictive")
				require.NoError(t, os.Mkdir(dir, 0o755))
				return dir
			},
			wantErr: true,
			errMsg:  "incorrect permissions",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dir := tc.setup(t)
			err := EnsureAtLeastSecureDir(dir)

			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					require.ErrorContains(t, err, tc.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestEnsureAtLeastSecureDirWithNestedPaths(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	nestedPath := filepath.Join(tempDir, "level1", "level2", "level3")

	err := EnsureAtLeastSecureDir(nestedPath)
	require.NoError(t, err)

	info, err := os.Stat(nestedPath)
	require.NoError(t, err)
	require.True(t, info.IsDir())
	require.True(t, isPermissionAcceptable(info.Mode().Perm(), perms.SecureDir))
}

func TestEnsureAtLeastSecureDirErrorMessages(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	tooOpen := filepath.Join(tempDir, "too-open")
	require.NoError(t, os.Mkdir(tooOpen, 0o755))

	err := EnsureAtLeastSecureDir(tooOpen)
	require.Error(t, err)
	expectedErr := fmt.Sprintf(
		"incorrect permissions for directory '%s' (0755, want 0700 or more restrictive)",
		tooOpen,
	)
	require.EqualError(t, err, expectedErr)
}

func TestEnsureAtLeastRegularDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		wantErr bool
		errMsg  string
	}{
		{
			name: "creates directory when it doesn't exist",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "new-regular-dir")
			},
			wantErr: false,
		},
		{
			name: "accepts directory with exact required permissions",
			setup: func(t *testing.T) string {
				dir := filepath.Join(t.TempDir(), "exact-perms")
				require.NoError(t, os.Mkdir(dir, perms.RegularDir))
				return dir
			},
			wantErr: false,
		},
		{
			name: "accepts directory with more restrictive permissions",
			setup: func(t *testing.T) string {
				dir := filepath.Join(t.TempDir(), "more-restrictive")
				require.NoError(t, os.Mkdir(dir, 0o700))
				return dir
			},
			wantErr: false,
		},
		{
			name: "rejects directory with less restrictive permissions",
			setup: func(t *testing.T) string {
				dir := filepath.Join(t.TempDir(), "less-restrictive")
				// NOTE: os.Mkdir applies the process umask, so passing 0o777 does not guarantee
				// world-writable permissions (e.g. 0o777 &^ 0o022 = 0o755 on most systems).
				// We explicitly chmod here to ensure the directory actually has 0o777 permissions
				// for the test to validate incorrect-permission handling correctly.
				require.NoError(t, os.Mkdir(dir, 0o755))
				require.NoError(t, os.Chmod(dir, 0o777))
				return dir
			},
			wantErr: true,
			errMsg:  "incorrect permissions",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dir := tc.setup(t)
			err := EnsureAtLeastRegularDir(dir)

			if tc.wantErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					require.ErrorContains(t, err, tc.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestEnsureAtLeastRegularDirWithNestedPaths(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	nestedPath := filepath.Join(tempDir, "level1", "level2", "level3")

	err := EnsureAtLeastRegularDir(nestedPath)
	require.NoError(t, err)

	info, err := os.Stat(nestedPath)
	require.NoError(t, err)
	require.True(t, info.IsDir())
	require.True(t, isPermissionAcceptable(info.Mode().Perm(), perms.RegularDir))
}

func TestEnsureAtLeastRegularDirErrorMessages(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	tooOpen := filepath.Join(tempDir, "too-open")
	// NOTE: os.Mkdir applies the process umask, so passing 0o777 does not guarantee
	// world-writable permissions (e.g. 0o777 &^ 0o022 = 0o755 on most systems).
	// We explicitly chmod here to ensure the directory actually has 0o777 permissions
	// for the test to validate incorrect-permission handling correctly.
	require.NoError(t, os.Mkdir(tooOpen, 0o755))
	require.NoError(t, os.Chmod(tooOpen, 0o777))

	err := EnsureAtLeastRegularDir(tooOpen)
	require.Error(t, err)
	require.ErrorContains(t, err, "incorrect permissions")
	require.ErrorContains(t, err, tooOpen)
}

func TestDiscoverExecutables(t *testing.T) {
	t.Parallel()

	t.Run("non-existent directory", func(t *testing.T) {
		t.Parallel()

		_, err := DiscoverExecutables("/nonexistent/path")
		require.Error(t, err)
	})

	t.Run("empty directory", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		executables, err := DiscoverExecutables(tempDir)
		require.NoError(t, err)
		require.Empty(t, executables)
	})

	t.Run("directory with executable files", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()

		// Create executable plugin.
		execPath := filepath.Join(tempDir, "test-plugin")
		err := os.WriteFile(execPath, []byte("#!/bin/sh\necho test"), 0o755)
		require.NoError(t, err)

		executables, err := DiscoverExecutables(tempDir)
		require.NoError(t, err)
		require.Len(t, executables, 1)
		require.Contains(t, executables, "test-plugin")
	})

	t.Run("skips non-executable files", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()

		// Create executable.
		execPath := filepath.Join(tempDir, "plugin")
		err := os.WriteFile(execPath, []byte("#!/bin/sh"), 0o755)
		require.NoError(t, err)

		// Create non-executable.
		nonExecPath := filepath.Join(tempDir, "readme.txt")
		err = os.WriteFile(nonExecPath, []byte("readme"), 0o644)
		require.NoError(t, err)

		executables, err := DiscoverExecutables(tempDir)
		require.NoError(t, err)
		require.Len(t, executables, 1)
		require.Contains(t, executables, "plugin")
		require.NotContains(t, executables, "readme.txt")
	})

	t.Run("skips hidden files", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()

		// Create visible executable.
		visiblePath := filepath.Join(tempDir, "visible-plugin")
		err := os.WriteFile(visiblePath, []byte("#!/bin/sh"), 0o755)
		require.NoError(t, err)

		// Create hidden executable.
		hiddenPath := filepath.Join(tempDir, ".hidden-plugin")
		err = os.WriteFile(hiddenPath, []byte("#!/bin/sh"), 0o755)
		require.NoError(t, err)

		executables, err := DiscoverExecutables(tempDir)
		require.NoError(t, err)
		require.Len(t, executables, 1)
		require.Contains(t, executables, "visible-plugin")
		require.NotContains(t, executables, ".hidden-plugin")
	})

	t.Run("skips subdirectories", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()

		// Create executable file.
		execPath := filepath.Join(tempDir, "plugin")
		err := os.WriteFile(execPath, []byte("#!/bin/sh"), 0o755)
		require.NoError(t, err)

		// Create subdirectory.
		subDir := filepath.Join(tempDir, "subdir")
		err = os.Mkdir(subDir, 0o755)
		require.NoError(t, err)

		executables, err := DiscoverExecutables(tempDir)
		require.NoError(t, err)
		require.Len(t, executables, 1)
		require.Contains(t, executables, "plugin")
	})
}

func TestDiscoverExecutablesWithPaths(t *testing.T) {
	t.Parallel()

	t.Run("filters by allowed list", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()

		// Create multiple executables.
		for _, name := range []string{"plugin1", "plugin2", "plugin3"} {
			path := filepath.Join(tempDir, name)
			err := os.WriteFile(path, []byte("#!/bin/sh"), 0o755)
			require.NoError(t, err)
		}

		allowed := map[string]struct{}{
			"plugin1": {},
			"plugin3": {},
		}

		executables, err := DiscoverExecutablesWithPaths(tempDir, allowed)
		require.NoError(t, err)
		require.Len(t, executables, 2)
		require.Contains(t, executables, "plugin1")
		require.Contains(t, executables, "plugin3")
		require.NotContains(t, executables, "plugin2")
		require.Equal(t, filepath.Join(tempDir, "plugin1"), executables["plugin1"])
		require.Equal(t, filepath.Join(tempDir, "plugin3"), executables["plugin3"])
	})

	t.Run("nil allowed list includes all executables", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()

		// Create executables.
		execPath := filepath.Join(tempDir, "plugin")
		err := os.WriteFile(execPath, []byte("#!/bin/sh"), 0o755)
		require.NoError(t, err)

		executables, err := DiscoverExecutablesWithPaths(tempDir, nil)
		require.NoError(t, err)
		require.Len(t, executables, 1)
		require.Contains(t, executables, "plugin")
		require.Equal(t, filepath.Join(tempDir, "plugin"), executables["plugin"])
	})
}
