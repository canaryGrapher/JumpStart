//go:build windows

package procman

import (
	"os/exec"
	"strconv"
	"syscall"
)

// shellCommand wraps a command line in the platform shell.
func shellCommand(command string) *exec.Cmd {
	cmd := exec.Command("cmd", "/C", command)
	return cmd
}

// setProcGroupAttrs starts the child in a new process group and without a
// console window popping up.
func setProcGroupAttrs(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.CreationFlags |= syscall.CREATE_NEW_PROCESS_GROUP
	cmd.SysProcAttr.HideWindow = true
}

// terminateTree stops the process tree rooted at pid. Windows has no
// SIGTERM equivalent for arbitrary trees, so taskkill /T is the practical
// graceful-ish first attempt (it sends WM_CLOSE to GUI apps and ends
// console children).
func terminateTree(pid int) {
	cmd := exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T")
	hideConsole(cmd)
	_ = cmd.Run()
}

// killTree force-kills the process tree rooted at pid.
func killTree(pid int) {
	cmd := exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T", "/F")
	hideConsole(cmd)
	_ = cmd.Run()
}

// hideConsole prevents helper invocations from flashing a console window.
func hideConsole(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
