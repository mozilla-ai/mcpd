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

	// EnvVarXDGCacheHome is the XDG Base Directory env var name for cache file.
	EnvVarXDGCacheHome = "XDG_CACHE_HOME"
)

// AppDirName returns the name of the application directory for use in user-specific operations where data is being written.
func AppDirName() string {
	return "mcpd"
}

// DiscoverExecutables scans a directory and returns a set of executable file names.
// Skips directories and hidden files (starting with ".").
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
// Only includes files present in the allowed set if provided (nil allowed means include all).
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
// Used for cache directories, data directories, and documentation.
func EnsureAtLeastRegularDir(path string) error {
	return ensureAtLeastDir(path, perms.RegularDir)
}

// EnsureAtLeastSecureDir creates a directory with secure permissions if it doesn't exist,
// and verifies that it has at least the required secure permissions if it already exists.
// It does not attempt to repair ownership or permissions: if they are wrong,
// it returns an error.
func EnsureAtLeastSecureDir(path string) error {
	return ensureAtLeastDir(path, perms.SecureDir)
}

// UserSpecificCacheDir returns the directory that should be used to store any user-specific cache files.
// It adheres to the XDG Base Directory Specification, respecting the XDG_CACHE_HOME environment variable.
// When XDG_CACHE_HOME is not set, it defaults to ~/.cache/mcpd/
// See: https://specifications.freedesktop.org/basedir-spec/latest/
func UserSpecificCacheDir() (string, error) {
	return userSpecificDir(EnvVarXDGCacheHome, ".cache")
}

// UserSpecificConfigDir returns the directory that should be used to store any user-specific configuration.
// It adheres to the XDG Base Directory Specification, respecting the XDG_CONFIG_HOME environment variable.
// When XDG_CONFIG_HOME is not set, it defaults to ~/.config/mcpd/
// See: https://specifications.freedesktop.org/basedir-spec/latest/
func UserSpecificConfigDir() (string, error) {
	return userSpecificDir(EnvVarXDGConfigHome, ".config")
}

// ensureAtLeastDir creates a directory with the specified permissions if it doesn't exist,
// and verifies that it has at least the required permissions if it already exists.
// It does not attempt to repair ownership or permissions: if they are wrong, it returns an error.
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
// "More restrictive" means: no permission bit set in actual that isn't also set in required.
func isPermissionAcceptable(actual, required os.FileMode) bool {
	// Check that actual doesn't grant any permissions that required doesn't grant.
	return (actual & ^required) == 0
}

// userSpecificDir returns a user-specific directory following XDG Base Directory Specification.
// It respects the given environment variable, falling back to homeDir/dir/AppDirName() if not set.
// The envVar must have XDG_ prefix to follow the specification.
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
