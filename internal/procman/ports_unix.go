//go:build !windows

package procman

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// portsForGroup asks lsof for listening TCP ports of a process group.
// Works on macOS and Linux.
func portsForGroup(pgid int) []int {
	cmd := exec.Command("lsof", "-a", "-g", strconv.Itoa(pgid),
		"-iTCP", "-sTCP:LISTEN", "-P", "-n")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	re := regexp.MustCompile(`:(\d{2,5})\s+\(LISTEN\)`)
	var ports []int
	for _, line := range strings.Split(string(out), "\n") {
		if m := re.FindStringSubmatch(line); m != nil {
			if p, err := strconv.Atoi(m[1]); err == nil {
				ports = append(ports, p)
			}
		}
	}
	return ports
}
