//go:build !windows

package files

import (
	"fmt"
	"os"
)

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
