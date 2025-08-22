package perms

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilePermissionConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		perm     os.FileMode
		expected os.FileMode
		octal    string
	}{
		{
			name:     "RegularFile has correct permissions",
			perm:     RegularFile,
			expected: 0o644,
			octal:    "0644",
		},
		{
			name:     "SecureFile has correct permissions",
			perm:     SecureFile,
			expected: 0o600,
			octal:    "0600",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.expected, tc.perm, "Permission constant should match expected octal value %s", tc.octal)
		})
	}
}

func TestDirectoryPermissionConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		perm     os.FileMode
		expected os.FileMode
		octal    string
	}{
		{
			name:     "RegularDir has correct permissions",
			perm:     RegularDir,
			expected: 0o755,
			octal:    "0755",
		},
		{
			name:     "SecureDir has correct permissions",
			perm:     SecureDir,
			expected: 0o700,
			octal:    "0700",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.expected, tc.perm, "Permission constant should match expected octal value %s", tc.octal)
		})
	}
}

func TestPermissionTypeInference(t *testing.T) {
	t.Parallel()

	// Ensure constants are of os.FileMode type.
	require.IsType(t, os.FileMode(0), RegularFile, "RegularFile should be of type os.FileMode")
	require.IsType(t, os.FileMode(0), SecureFile, "SecureFile should be of type os.FileMode")
	require.IsType(t, os.FileMode(0), RegularDir, "RegularDir should be of type os.FileMode")
	require.IsType(t, os.FileMode(0), SecureDir, "SecureDir should be of type os.FileMode")
}

func TestPermissionBitMasks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		perm        os.FileMode
		ownerRead   bool
		ownerWrite  bool
		ownerExec   bool
		groupRead   bool
		groupWrite  bool
		groupExec   bool
		othersRead  bool
		othersWrite bool
		othersExec  bool
	}{
		{
			name:        "RegularFile permissions breakdown",
			perm:        RegularFile,
			ownerRead:   true,
			ownerWrite:  true,
			ownerExec:   false,
			groupRead:   true,
			groupWrite:  false,
			groupExec:   false,
			othersRead:  true,
			othersWrite: false,
			othersExec:  false,
		},
		{
			name:        "SecureFile permissions breakdown",
			perm:        SecureFile,
			ownerRead:   true,
			ownerWrite:  true,
			ownerExec:   false,
			groupRead:   false,
			groupWrite:  false,
			groupExec:   false,
			othersRead:  false,
			othersWrite: false,
			othersExec:  false,
		},
		{
			name:        "RegularDir permissions breakdown",
			perm:        RegularDir,
			ownerRead:   true,
			ownerWrite:  true,
			ownerExec:   true,
			groupRead:   true,
			groupWrite:  false,
			groupExec:   true,
			othersRead:  true,
			othersWrite: false,
			othersExec:  true,
		},
		{
			name:        "SecureDir permissions breakdown",
			perm:        SecureDir,
			ownerRead:   true,
			ownerWrite:  true,
			ownerExec:   true,
			groupRead:   false,
			groupWrite:  false,
			groupExec:   false,
			othersRead:  false,
			othersWrite: false,
			othersExec:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Check owner permissions.
			require.Equal(t, tc.ownerRead, tc.perm&0o400 != 0, "Owner read permission mismatch")
			require.Equal(t, tc.ownerWrite, tc.perm&0o200 != 0, "Owner write permission mismatch")
			require.Equal(t, tc.ownerExec, tc.perm&0o100 != 0, "Owner execute permission mismatch")

			// Check group permissions.
			require.Equal(t, tc.groupRead, tc.perm&0o040 != 0, "Group read permission mismatch")
			require.Equal(t, tc.groupWrite, tc.perm&0o020 != 0, "Group write permission mismatch")
			require.Equal(t, tc.groupExec, tc.perm&0o010 != 0, "Group execute permission mismatch")

			// Check others permissions.
			require.Equal(t, tc.othersRead, tc.perm&0o004 != 0, "Others read permission mismatch")
			require.Equal(t, tc.othersWrite, tc.perm&0o002 != 0, "Others write permission mismatch")
			require.Equal(t, tc.othersExec, tc.perm&0o001 != 0, "Others execute permission mismatch")
		})
	}
}

func TestSecurityClassifications(t *testing.T) {
	t.Parallel()

	// Test that secure permissions are more restrictive than regular permissions.
	t.Run("SecureFile is more restrictive than RegularFile", func(t *testing.T) {
		t.Parallel()

		// SecureFile should have fewer permission bits set.
		require.True(
			t,
			SecureFile&RegularFile == SecureFile,
			"SecureFile should be a subset of RegularFile permissions",
		)
		require.True(t, SecureFile < RegularFile, "SecureFile should be numerically less than RegularFile")
	})

	t.Run("SecureDir is more restrictive than RegularDir", func(t *testing.T) {
		t.Parallel()

		// SecureDir should have fewer permission bits set.
		require.True(t, SecureDir&RegularDir == SecureDir, "SecureDir should be a subset of RegularDir permissions")
		require.True(t, SecureDir < RegularDir, "SecureDir should be numerically less than RegularDir")
	})
}
