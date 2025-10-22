package files

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mozilla-ai/mcpd/v2/internal/perms"
)

const (
	// EnvVarXDGConfigHome is the XDG Base Directory env var name for config files.
	EnvVarXDGConfigHome = "XDG_CONFIG_HOME"

	// EnvVarXDGCacheHome is the XDG Base Directory env var name for cache files.
	EnvVarXDGCacheHome = "XDG_CACHE_HOME"
)

// AppDirName returns the application directory name used for user-specific configuration and cache paths ("mcpd").
func AppDirName() string {
	return "mcpd"
}

// DiscoverExecutables scans a directory and returns a set of executable file names.
// DiscoverExecutables scans dir and returns the set of executable file names found.
// It skips directories and files whose names start with ".". A file is considered
// executable when any of the user/group/other execute permission bits are set.
// On error it returns nil and the underlying error.
func DiscoverExecutables(dir string) (map[string]struct{}, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	executables := make(map[string]struct{})
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("reading file info for %s: %w", entry.Name(), err)
		}

		// Check for execute permission (0o111 = user/group/other execute bits).
		if info.Mode()&0o111 != 0 {
			executables[entry.Name()] = struct{}{}
		}
	}

	return executables, nil
}

// DiscoverExecutablesWithPaths scans a directory and returns a map of executable names to their full paths.
// Skips directories and hidden files (starting with ".").
// DiscoverExecutablesWithPaths scans dir and returns a map from executable file name to its full path,
// optionally restricted to names present in allowed.
//
// It ignores directory entries and hidden files (names starting with '.'). When allowed is non-nil,
// only entries whose names are keys in allowed are considered. A file is included if any execute bit
// is set in its mode. Returns an error if the directory cannot be read or if file information cannot be retrieved.
func DiscoverExecutablesWithPaths(dir string, allowed map[string]struct{}) (map[string]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	executables := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		// Skip if not in allowed list (when allowed list is provided).
		if allowed != nil {
			if _, ok := allowed[entry.Name()]; !ok {
				continue
			}
		}

		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("reading file info for %s: %w", entry.Name(), err)
		}

		// Check for execute permission (0o111 = user/group/other execute bits).
		if info.Mode()&0o111 != 0 {
			fullPath := filepath.Join(dir, entry.Name())
			executables[entry.Name()] = fullPath
		}
	}

	return executables, nil
}

// EnsureAtLeastRegularDir creates a directory with standard permissions if it doesn't exist,
// and verifies that it has at least the required regular permissions if it already exists.
// It does not attempt to repair ownership or permissions: if they are wrong, it returns an error.
// EnsureAtLeastRegularDir ensures the directory at path exists and has at least the standard regular directory permissions.
// It creates the directory with regular permissions if it does not exist, or verifies existing permissions meet the required regular permissions; returns an error if creation or the permission check fails.
func EnsureAtLeastRegularDir(path string) error {
	return ensureAtLeastDir(path, perms.RegularDir)
}

// EnsureAtLeastSecureDir creates a directory with secure permissions if it doesn't exist,
// and verifies that it has at least the required secure permissions if it already exists.
// It does not attempt to repair ownership or permissions: if they are wrong,
// EnsureAtLeastSecureDir ensures a directory exists at path with at least the package's secure directory permissions.
// If the directory is missing it will be created; if it exists its permissions must be no less restrictive than the secure policy.
func EnsureAtLeastSecureDir(path string) error {
	return ensureAtLeastDir(path, perms.SecureDir)
}

// UserSpecificCacheDir returns the directory that should be used to store any user-specific cache files.
// It adheres to the XDG Base Directory Specification, respecting the XDG_CACHE_HOME environment variable.
// When XDG_CACHE_HOME is not set, it defaults to ~/.cache/mcpd/
// UserSpecificCacheDir returns the user-specific cache directory path for the application
// following the XDG Base Directory Specification.
// If the XDG_CACHE_HOME environment variable is set and non-empty, its value is joined
// with the application directory name; otherwise the function falls back to
// $HOME/.cache/<appdir>. An error is returned if the user's home directory cannot be determined.
func UserSpecificCacheDir() (string, error) {
	return userSpecificDir(EnvVarXDGCacheHome, ".cache")
}

// UserSpecificConfigDir returns the directory that should be used to store any user-specific configuration.
// It adheres to the XDG Base Directory Specification, respecting the XDG_CONFIG_HOME environment variable.
// When XDG_CONFIG_HOME is not set, it defaults to ~/.config/mcpd/
// UserSpecificConfigDir returns the user-specific configuration directory for this application
// following the XDG Base Directory Specification.
// If the XDG_CONFIG_HOME environment variable is set and non-empty, its value is used; otherwise
// the directory $HOME/.config is used. The returned path is the chosen base directory joined with
// AppDirName().
// An error is returned if the XDG environment variable name is invalid or the current user's home
// directory cannot be determined.
func UserSpecificConfigDir() (string, error) {
	return userSpecificDir(EnvVarXDGConfigHome, ".config")
}

// ensureAtLeastDir creates a directory with the specified permissions if it doesn't exist,
// and verifies that it has at least the required permissions if it already exists.
// ensureAtLeastDir ensures a directory exists at path and that its permission bits are no less restrictive than perm.
// If the directory does not exist it is created with perm; if creating, statting, or the permission check fails, an error is returned.
// It does not attempt to change ownership or otherwise repair permissions.
func ensureAtLeastDir(path string, perm os.FileMode) error {
	if err := os.MkdirAll(path, perm); err != nil {
		return fmt.Errorf("could not ensure directory exists for '%s': %w", path, err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("could not stat directory '%s': %w", path, err)
	}

	if !isPermissionAcceptable(info.Mode().Perm(), perm) {
		return fmt.Errorf(
			"incorrect permissions for directory '%s' (%#o, want %#o or more restrictive)",
			path, info.Mode().Perm(),
			perm,
		)
	}

	return nil
}

// isPermissionAcceptable checks if the actual permissions are acceptable for the required permissions.
// It returns true if the actual permissions are equal to or more restrictive than required.
// isPermissionAcceptable reports whether the actual file mode is equal to or more restrictive than the required mode.
// It returns true if no permission bit is set in actual that is not also set in required.
func isPermissionAcceptable(actual, required os.FileMode) bool {
	// Check that actual doesn't grant any permissions that required doesn't grant.
	return (actual & ^required) == 0
}

// userSpecificDir returns a user-specific directory following XDG Base Directory Specification.
// It respects the given environment variable, falling back to homeDir/dir/AppDirName() if not set.
// userSpecificDir returns the user-specific directory path for the application
// following the XDG Base Directory Specification.
//
// envVar must start with the "XDG_" prefix; if the corresponding environment
// variable is set and non-empty its value is used as the base directory.
// Otherwise the function falls back to the current user's home directory and
// appends dir and the application directory name.
//
// It returns an error if envVar does not start with "XDG_" or if the user's
// home directory cannot be determined.
func userSpecificDir(envVar string, dir string) (string, error) {
	envVar = strings.TrimSpace(envVar)
	// Validate that the environment variable follows XDG naming convention.
	if !strings.HasPrefix(envVar, "XDG_") {
		return "", fmt.Errorf(
			"environment variable '%s' does not follow XDG Base Directory Specification",
			envVar,
		)
	}

	// If the relevant environment variable is present and configured, then use it.
	if ch, ok := os.LookupEnv(envVar); ok && strings.TrimSpace(ch) != "" {
		home := strings.TrimSpace(ch)
		return filepath.Join(home, AppDirName()), nil
	}

	// Attempt to locate the home directory for the current user and return the path that follows the spec.
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	return filepath.Join(homeDir, dir, AppDirName()), nil
}