package procman

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var portPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)port[\s:=]+(\d{2,5})`),
	regexp.MustCompile(`(?:localhost|127\.0\.0\.1|0\.0\.0\.0|\[::1?\]):(\d{2,5})`),
}

// portsFromLine extracts port numbers mentioned in a log line.
func portsFromLine(line string) []int {
	var out []int
	for _, re := range portPatterns {
		for _, m := range re.FindAllStringSubmatch(line, -1) {
			if p, err := strconv.Atoi(m[1]); err == nil && p > 0 && p < 65536 {
				out = append(out, p)
			}
		}
	}
	return out
}

// portsFromLsof asks lsof for listening TCP ports of a process group.
// Works on macOS and Linux.
func portsFromLsof(pgid int) []int {
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
