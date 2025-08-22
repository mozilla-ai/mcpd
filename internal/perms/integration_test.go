package perms

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestFileCreationPermissions verifies that files created with perms constants
// have the correct permissions on the filesystem.
func TestFileCreationPermissions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		perm     os.FileMode
		expected os.FileMode
	}{
		{
			name:     "RegularFile creates file with 0644",
			perm:     RegularFile,
			expected: 0o644,
		},
		{
			name:     "SecureFile creates file with 0600",
			perm:     SecureFile,
			expected: 0o600,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			filePath := filepath.Join(tempDir, "test-file")

			// Create file using the permission constant.
			err := os.WriteFile(filePath, []byte("test content"), tc.perm)
			require.NoError(t, err)

			// Verify actual permissions on filesystem.
			info, err := os.Stat(filePath)
			require.NoError(t, err)
			require.Equal(t, tc.expected, info.Mode().Perm(), "File permissions should match expected value")
		})
	}
}

// TestDirectoryCreationPermissions verifies that directories created with perms constants
// have the correct permissions on the filesystem.
func TestDirectoryCreationPermissions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		perm     os.FileMode
		expected os.FileMode
	}{
		{
			name:     "RegularDir creates directory with 0755",
			perm:     RegularDir,
			expected: 0o755,
		},
		{
			name:     "SecureDir creates directory with 0700",
			perm:     SecureDir,
			expected: 0o700,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			dirPath := filepath.Join(tempDir, "test-dir")

			// Create directory using the permission constant.
			err := os.MkdirAll(dirPath, tc.perm)
			require.NoError(t, err)

			// Verify actual permissions on filesystem.
			info, err := os.Stat(dirPath)
			require.NoError(t, err)
			require.True(t, info.IsDir())
			require.Equal(t, tc.expected, info.Mode().Perm(), "Directory permissions should match expected value")
		})
	}
}

// TestOpenFilePermissions verifies that files opened with perms constants
// have the correct permissions on the filesystem.
func TestOpenFilePermissions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		perm     os.FileMode
		expected os.FileMode
	}{
		{
			name:     "RegularFile with OpenFile has 0644",
			perm:     RegularFile,
			expected: 0o644,
		},
		{
			name:     "SecureFile with OpenFile has 0600",
			perm:     SecureFile,
			expected: 0o600,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tempDir := t.TempDir()
			filePath := filepath.Join(tempDir, "test-file")

			// Open file for creation using the permission constant.
			f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, tc.perm)
			require.NoError(t, err)

			_, err = f.WriteString("test content")
			require.NoError(t, err)

			err = f.Close()
			require.NoError(t, err)

			// Verify actual permissions on filesystem.
			info, err := os.Stat(filePath)
			require.NoError(t, err)
			require.Equal(t, tc.expected, info.Mode().Perm(), "File permissions should match expected value")
		})
	}
}

// TestPermissionInheritance verifies that created files inherit the correct
// permissions regardless of parent directory permissions.
func TestPermissionInheritance(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	// Create parent directory with different permissions.
	parentDir := filepath.Join(tempDir, "parent")
	err := os.MkdirAll(parentDir, 0o777) // Highly permissive parent
	require.NoError(t, err)

	// Create secure file in permissive parent directory.
	secureFilePath := filepath.Join(parentDir, "secure-file")
	err = os.WriteFile(secureFilePath, []byte("secure content"), SecureFile)
	require.NoError(t, err)

	// Verify secure file has correct permissions despite permissive parent.
	info, err := os.Stat(secureFilePath)
	require.NoError(t, err)
	require.Equal(
		t,
		SecureFile,
		info.Mode().Perm(),
		"Secure file should have 0600 permissions despite parent directory permissions",
	)

	// Create secure directory in permissive parent directory.
	secureChildDir := filepath.Join(parentDir, "secure-child")
	err = os.MkdirAll(secureChildDir, SecureDir)
	require.NoError(t, err)

	// Verify secure directory has correct permissions.
	info, err = os.Stat(secureChildDir)
	require.NoError(t, err)
	require.True(t, info.IsDir())
	require.Equal(t, SecureDir, info.Mode().Perm(), "Secure directory should have 0700 permissions")
}

// TestPermissionNames verifies that the permission constants have appropriate names
// that reflect their security classification.
func TestPermissionNames(t *testing.T) {
	t.Parallel()

	// Test naming convention consistency.
	t.Run("secure permissions are more restrictive than regular", func(t *testing.T) {
		t.Parallel()

		require.True(
			t,
			SecureFile < RegularFile,
			"SecureFile should be more restrictive (lower value) than RegularFile",
		)
		require.True(t, SecureDir < RegularDir, "SecureDir should be more restrictive (lower value) than RegularDir")
	})

	t.Run("file permissions don't include execute bit", func(t *testing.T) {
		t.Parallel()

		// Files should not have execute permissions for owner, group, or others.
		require.Equal(t, os.FileMode(0), RegularFile&0o111, "RegularFile should not have execute permissions")
		require.Equal(t, os.FileMode(0), SecureFile&0o111, "SecureFile should not have execute permissions")
	})

	t.Run("directory permissions include execute bit for accessibility", func(t *testing.T) {
		t.Parallel()

		// Directories need execute bit for traversal.
		require.NotEqual(t, os.FileMode(0), RegularDir&0o100, "RegularDir should have owner execute permission")
		require.NotEqual(t, os.FileMode(0), SecureDir&0o100, "SecureDir should have owner execute permission")
	})
}

// TestPermissionStringRepresentation verifies that permission constants
// can be properly formatted as strings for logging and error messages.
func TestPermissionStringRepresentation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		perm             os.FileMode
		expectedSymbolic string
		expectedOctal    string
	}{
		{
			name:             "RegularFile string representation",
			perm:             RegularFile,
			expectedSymbolic: "-rw-r--r--",
			expectedOctal:    "644",
		},
		{
			name:             "SecureFile string representation",
			perm:             SecureFile,
			expectedSymbolic: "-rw-------",
			expectedOctal:    "600",
		},
		{
			name:             "RegularDir string representation",
			perm:             RegularDir,
			expectedSymbolic: "rwxr-xr-x",
			expectedOctal:    "755",
		},
		{
			name:             "SecureDir string representation",
			perm:             SecureDir,
			expectedSymbolic: "rwx------",
			expectedOctal:    "700",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Test symbolic representation (os.FileMode.String()).
			formatted := tc.perm.String()
			require.Contains(t, formatted, tc.expectedSymbolic,
				"Permission %s symbolic format should contain %s, got: %s", tc.name, tc.expectedSymbolic, formatted)

			// Test octal formatting (%#o).
			octalFormatted := fmt.Sprintf("%#o", tc.perm)
			require.Contains(t, octalFormatted, tc.expectedOctal,
				"Permission %s octal format should contain %s, got: %s", tc.name, tc.expectedOctal, octalFormatted)
		})
	}
}
