//go:build unix

package plugin

import (
	"os/exec"
	"syscall"
)

// setProcessGroup configures cmd to run in its own process group so that,
// on cleanup, killProcessGroup can also terminate descendants that remain
// in that group, not just the direct child.
func setProcessGroup(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// killProcessGroup sends SIGKILL to the process group led by cmd's process,
// falling back to killing just the process if the group signal fails (e.g.
// setProcessGroup was never applied, or the group is already gone).
//
// This signals a process group ID, not a handle to the plugin's specific
// group: once the leader has been reaped and the group is empty, that ID
// can be reused, and a descendant that calls setsid or setpgid leaves the
// group and is never reached by this call either way. Real containment
// needs an OS-level handle - pidfds or cgroups on Linux, Job Objects on
// Windows - and is tracked as a follow-up rather than attempted here.
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err == nil {
		return nil
	}
	return cmd.Process.Kill()
}
