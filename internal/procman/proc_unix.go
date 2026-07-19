//go:build !windows

package procman

import (
	"os/exec"
	"syscall"
)

// shellCommand wraps a command line in the platform shell.
func shellCommand(command string) *exec.Cmd {
	return exec.Command("/bin/sh", "-c", command)
}

// setProcGroupAttrs puts the child in its own process group so the whole
// tree can be signalled at once.
func setProcGroupAttrs(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// terminateTree asks the process group to shut down gracefully.
func terminateTree(pgid int) {
	_ = syscall.Kill(-pgid, syscall.SIGTERM)
}

// killTree force-kills the process group.
func killTree(pgid int) {
	_ = syscall.Kill(-pgid, syscall.SIGKILL)
}
