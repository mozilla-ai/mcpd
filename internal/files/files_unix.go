//go:build !windows

package files

import (
	"fmt"
	"os"
)

// executableRank reports whether the file described by name and info can be executed on this platform,
// along with a rank used to break ties between candidates that share the same registered name
// (lower is preferred). On Unix-like systems a file is executable when any execute bit is set,
// and names are never rewritten, so the rank is always zero.
func executableRank(_ string, info os.FileInfo) (int, bool) {
	if !info.Mode().IsRegular() {
		return 0, false
	}

	return 0, info.Mode().Perm()&0o111 != 0
}

// executableBaseName returns the name a discovered executable is registered under.
// On Unix-like systems this is the file name itself.
func executableBaseName(name string) string {
	return name
}

// validateDirPermissions verifies that the directory at path has permissions equal to,
// or more restrictive than, required.
func validateDirPermissions(path string, info os.FileInfo, required os.FileMode) error {
	if !isPermissionAcceptable(info.Mode().Perm(), required) {
		return fmt.Errorf(
			"incorrect permissions for directory '%s' (%#o, want %#o or more restrictive)",
			path, info.Mode().Perm(),
			required,
		)
	}

	return nil
}
