//go:build unix

package plugin

import (
	"os/exec"
	"syscall"
)

// setProcessGroup configures cmd to run in its own process group so that,
// on cleanup, we can terminate the plugin binary along with any descendant
// processes it spawns rather than only the direct child.
func setProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// killProcessGroup sends SIGKILL to the process group led by cmd's process,
// falling back to killing just the process if the group signal fails (e.g.
// setProcessGroup was never applied, or the group is already gone).
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err == nil {
		return nil
	}
	return cmd.Process.Kill()
}
