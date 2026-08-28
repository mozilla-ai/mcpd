//go:build windows

package plugin

import "os/exec"

// setProcessGroup is a no-op on Windows: process-group termination is
// implemented on Unix only, so cleanup falls back to killing the direct
// process.
func setProcessGroup(cmd *exec.Cmd) {}

// killProcessGroup kills the plugin's direct process. Windows has no
// equivalent of a POSIX process group signal here, so descendants that
// inherit stdout/stderr are not separately terminated.
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
