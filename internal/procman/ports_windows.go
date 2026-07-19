//go:build windows

package procman

import (
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// portsForGroup returns listening TCP ports owned by the process tree
// rooted at pid, by joining `netstat -ano` (port -> owning PID) with the
// tree of descendant PIDs.
func portsForGroup(pid int) []int {
	pids := descendantPIDs(pid)
	out, err := runHidden("netstat", "-ano")
	if err != nil {
		return nil
	}
	var ports []int
	seen := map[int]bool{}
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(line)
		// Proto LocalAddress ForeignAddress State PID
		if len(f) < 5 || f[0] != "TCP" || f[3] != "LISTENING" {
			continue
		}
		owner, err := strconv.Atoi(f[4])
		if err != nil || !pids[owner] {
			continue
		}
		addr := f[1]
		i := strings.LastIndex(addr, ":")
		if i < 0 {
			continue
		}
		if p, err := strconv.Atoi(addr[i+1:]); err == nil && p > 0 && !seen[p] {
			seen[p] = true
			ports = append(ports, p)
		}
	}
	return ports
}

// descendantPIDs returns the set {root} plus all transitive children,
// from a single CIM process listing.
func descendantPIDs(root int) map[int]bool {
	set := map[int]bool{root: true}
	out, err := runHidden("powershell", "-NoProfile", "-Command",
		"Get-CimInstance Win32_Process | ForEach-Object { \"$($_.ProcessId) $($_.ParentProcessId)\" }")
	if err != nil {
		return set
	}
	children := map[int][]int{}
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(strings.TrimSpace(line))
		if len(f) != 2 {
			continue
		}
		pid, err1 := strconv.Atoi(f[0])
		ppid, err2 := strconv.Atoi(f[1])
		if err1 != nil || err2 != nil {
			continue
		}
		children[ppid] = append(children[ppid], pid)
	}
	queue := []int{root}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, c := range children[cur] {
			if !set[c] {
				set[c] = true
				queue = append(queue, c)
			}
		}
	}
	return set
}

func runHidden(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	return string(out), err
}
