// Package perms provides centralized file and directory permission constants
// for consistent security practices across the mcpd codebase.
package perms

import "os"

// File permission constants for different security contexts.
const (
	// RegularFile permissions for standard files (configuration, logs, exports).
	// Mode 0644: owner read/write, group read, others read.
	RegularFile os.FileMode = 0o644

	// SecureFile permissions for sensitive files (execution context, credentials).
	// Mode 0600: owner read/write only, no group or other access.
	SecureFile os.FileMode = 0o600
)

// Directory permission constants for different security contexts.
const (
	// RegularDir permissions for standard directories (cache, data, documentation).
	// Mode 0755: owner read/write/execute, group read/execute, others read/execute.
	RegularDir os.FileMode = 0o755

	// SecureDir permissions for sensitive directories (execution context, private data).
	// Mode 0700: owner read/write/execute only, no group or other access.
	SecureDir os.FileMode = 0o700
)
