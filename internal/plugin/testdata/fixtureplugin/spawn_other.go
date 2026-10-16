//go:build !unix

package main

import "os/exec"

// setsid is a no-op outside unix: modeHealthyWithEscapedDescendant is only
// exercised by the unix-only integration tests, but this fixture package
// still needs to build wherever the go tool can reach it.
func setsid(cmd *exec.Cmd) {}
