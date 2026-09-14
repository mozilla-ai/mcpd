// Package filestest provides test helpers for creating filesystem fixtures that behave the same
// way on every platform.
package filestest

import "runtime"

// ExecutableFileName returns the file name to give a test fixture that must be discovered as an
// executable called name on the current platform.
// name is the bare name a plugin would be configured with and must not carry an executable
// extension; on Windows ".exe" is appended, elsewhere the name is returned unchanged.
func ExecutableFileName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}

	return name
}
