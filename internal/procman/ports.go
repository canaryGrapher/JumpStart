package procman

import (
	"regexp"
	"strconv"
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

// portsForGroup returns the listening TCP ports owned by the process
// group/tree rooted at pid. Implemented per platform in ports_unix.go
// (lsof) and ports_windows.go (netstat).
