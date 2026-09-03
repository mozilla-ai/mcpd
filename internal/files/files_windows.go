//go:build windows

package files

import "os"

// validateDirPermissions accepts every directory on Windows; no access check is performed.
//
// TODO(#302): enforce an owner-only DACL for secure directories instead of accepting everything.
//
// Windows has no POSIX mode bits to compare: Go synthesizes 0777 for every directory regardless of
// the mode requested at creation, so the Unix check can never pass here. Access is governed by the
// NTFS ACL the directory inherits from its parent, which this function does not inspect.
func validateDirPermissions(_ string, _ os.FileInfo, _ os.FileMode) error {
	return nil
}
