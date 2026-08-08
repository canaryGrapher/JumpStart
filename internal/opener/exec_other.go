//go:build !windows

package opener

import "os/exec"

// hideConsole is a no-op outside Windows: nothing here spawns a console host.
func hideConsole(*exec.Cmd) {}
