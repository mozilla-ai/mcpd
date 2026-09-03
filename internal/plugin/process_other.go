//go:build !unix && !windows

package plugin

import "os/exec"

// setProcessGroup is a no-op on platforms without POSIX process groups
// (for example js/wasm or plan9); cleanup falls back to killing the direct
// process. Nothing shipped targets these platforms, but the package should
// still compile everywhere the go tool can.
func setProcessGroup(cmd *exec.Cmd) {}

// killProcessGroup kills the plugin's direct process only, since there is
// no process-group signal to send on these platforms.
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
