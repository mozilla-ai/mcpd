//go:build windows

package files

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDiscoverExecutables_Windows verifies the PATHEXT-based discovery rules that only apply on Windows.
func TestDiscoverExecutables_Windows(t *testing.T) {
	// Not parallel: PATHEXT is process-wide and is modified by a subtest via t.Setenv.

	t.Run("uses PATHEXT extensions and strips them from the name", func(t *testing.T) {
		tempDir := t.TempDir()

		for _, name := range []string{"exe-plugin.exe", "cmd-plugin.CMD", "bat-plugin.bat", "readme.txt", "no-ext"} {
			require.NoError(t, os.WriteFile(filepath.Join(tempDir, name), []byte("binary"), 0o755))
		}

		executables, err := DiscoverExecutablesWithPaths(tempDir, nil)
		require.NoError(t, err)
		require.Len(t, executables, 3)
		require.Equal(t, filepath.Join(tempDir, "exe-plugin.exe"), executables["exe-plugin"])
		require.Equal(t, filepath.Join(tempDir, "cmd-plugin.CMD"), executables["cmd-plugin"])
		require.Equal(t, filepath.Join(tempDir, "bat-plugin.bat"), executables["bat-plugin"])
		require.NotContains(t, executables, "readme")
		require.NotContains(t, executables, "readme.txt")
		require.NotContains(t, executables, "no-ext")
	})

	t.Run("prefers the extension listed first in PATHEXT", func(t *testing.T) {
		t.Setenv("PATHEXT", ".COM;.EXE;.BAT;.CMD")
		tempDir := t.TempDir()

		// Sorted directory order would put plugin.bat first; PATHEXT order must win.
		for _, name := range []string{"plugin.bat", "plugin.exe", "plugin.cmd"} {
			require.NoError(t, os.WriteFile(filepath.Join(tempDir, name), []byte("binary"), 0o755))
		}

		executables, err := DiscoverExecutablesWithPaths(tempDir, nil)
		require.NoError(t, err)
		require.Len(t, executables, 1)
		require.Equal(t, filepath.Join(tempDir, "plugin.exe"), executables["plugin"])
	})

	t.Run("allowed list matches the extension-less name", func(t *testing.T) {
		tempDir := t.TempDir()

		for _, name := range []string{"plugin1.exe", "plugin2.exe"} {
			require.NoError(t, os.WriteFile(filepath.Join(tempDir, name), []byte("binary"), 0o755))
		}

		executables, err := DiscoverExecutablesWithPaths(tempDir, map[string]struct{}{"plugin2": {}})
		require.NoError(t, err)
		require.Len(t, executables, 1)
		require.Equal(t, filepath.Join(tempDir, "plugin2.exe"), executables["plugin2"])
	})

	t.Run("falls back to default extensions when PATHEXT is empty", func(t *testing.T) {
		t.Setenv("PATHEXT", "")
		tempDir := t.TempDir()

		require.NoError(t, os.WriteFile(filepath.Join(tempDir, "plugin.exe"), []byte("binary"), 0o755))

		executables, err := DiscoverExecutables(tempDir)
		require.NoError(t, err)
		require.Contains(t, executables, "plugin")
	})
}
