// Package permstest provides test helpers for asserting on-disk file and directory permissions
// in a way that is portable across platforms.
package permstest

import (
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// Enforced reports whether the current platform enforces POSIX permission mode bits.
// Windows has no such bits: Go reports 0666 for every file and 0777 for every directory
// regardless of the mode requested at creation.
func Enforced() bool {
	return runtime.GOOS != "windows"
}

// SkipWithoutPOSIXPermissions skips the test when the platform does not enforce POSIX mode bits.
// Use it for tests whose premise cannot be expressed without them, such as creating a
// deliberately too-permissive directory.
func SkipWithoutPOSIXPermissions(t *testing.T) {
	t.Helper()

	if !Enforced() {
		t.Skip("POSIX permissions are not enforced on this platform")
	}
}

// RequireMode asserts that the on-disk permission bits of info match expected.
// On platforms that do not enforce POSIX mode bits the assertion is not applied,
// so the surrounding test still verifies that the file or directory was created.
func RequireMode(t *testing.T, expected os.FileMode, info os.FileInfo, msgAndArgs ...any) {
	t.Helper()

	if !Enforced() {
		return
	}

	require.Equal(t, expected, info.Mode().Perm(), msgAndArgs...)
}
