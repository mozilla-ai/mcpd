//go:build windows

package files

import (
	"os"
	"path/filepath"
	"strings"
)

// defaultPathExt mirrors the Windows default for the PATHEXT environment variable.
const defaultPathExt = ".COM;.EXE;.BAT;.CMD"

// executableRank reports whether the file described by name and info can be executed on this platform,
// along with a rank used to break ties between candidates that share the same registered name
// (lower is preferred).
//
// Windows has no execute permission bit: Go reports every regular file as 0666, so the mode
// cannot be used. Instead a file is executable when its extension appears in PATHEXT
// (falling back to the Windows default of .COM;.EXE;.BAT;.CMD), matching the rules used by
// the shell and by exec.LookPath. The rank is the extension's position in PATHEXT, so
// "plugin.exe" is preferred over "plugin.cmd" when both are present.
func executableRank(name string, info os.FileInfo) (int, bool) {
	if !info.Mode().IsRegular() {
		return 0, false
	}

	ext := strings.ToLower(filepath.Ext(name))
	if ext == "" {
		return 0, false
	}

	for i, candidate := range executableExtensions() {
		if ext == candidate {
			return i, true
		}
	}

	return 0, false
}

// executableBaseName returns the name a discovered executable is registered under.
// On Windows the executable extension is stripped, so a configured plugin named "my-plugin"
// matches "my-plugin.exe" on disk.
func executableBaseName(name string) string {
	return strings.TrimSuffix(name, filepath.Ext(name))
}

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

// executableExtensions returns the lower-cased executable extensions from PATHEXT, in order of preference.
func executableExtensions() []string {
	pathExt := strings.TrimSpace(os.Getenv("PATHEXT"))
	if pathExt == "" {
		pathExt = defaultPathExt
	}

	parts := strings.Split(pathExt, ";")
	exts := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.ToLower(strings.TrimSpace(part))
		if part == "" || !strings.HasPrefix(part, ".") {
			continue
		}
		exts = append(exts, part)
	}

	return exts
}
