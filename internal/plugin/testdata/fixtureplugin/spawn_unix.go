//go:build unix

package main

import (
	"os/exec"
	"syscall"
)

// setsid configures cmd to start in a new session and process group of its
// own, detached from its parent's. This is how
// modeHealthyWithEscapedDescendant simulates a descendant that has left the
// plugin's process group on its own (via setsid or setpgid): killProcessGroup
// signals the plugin's process group, and a descendant that isn't in it any
// more is not reached, even though it still holds the plugin's
// stdout/stderr open.
func setsid(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setsid = true
}
