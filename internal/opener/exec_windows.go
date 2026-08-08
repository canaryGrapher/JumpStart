//go:build windows

package opener

import (
	"os/exec"
	"syscall"
)

// hideConsole stops a console window from flashing when we go through
// cmd.exe. JumpStart is a GUI binary, so any console child gets its own
// window unless it is suppressed.
func hideConsole(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
}
