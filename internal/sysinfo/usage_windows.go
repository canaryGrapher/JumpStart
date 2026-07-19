//go:build windows

package sysinfo

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

// snapshot lists processes once via CIM and attributes working-set memory
// to the managed process tree each PID belongs to. Per-process CPU percent
// is not cheaply available on Windows, so process CPU is reported as 0 and
// system CPU comes from the processor load average.
func snapshot(roots map[string]int) (SystemUsage, map[string]ProcUsage) {
	rows := cimRows()

	children := map[int][]int{}
	memByPID := map[int]float64{}
	for _, r := range rows {
		children[r.ppid] = append(children[r.ppid], r.pid)
		memByPID[r.pid] = r.wsKB / 1024
	}

	procs := map[string]ProcUsage{}
	for id, root := range roots {
		u := ProcUsage{}
		queue := []int{root}
		seen := map[int]bool{root: true}
		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]
			u.MemMB += memByPID[cur]
			for _, c := range children[cur] {
				if !seen[c] {
					seen[c] = true
					queue = append(queue, c)
				}
			}
		}
		procs[id] = u
	}

	sys := SystemUsage{NumCPU: runtime.NumCPU(), CPU: cpuLoadPercent()}
	sys.UsedMemMB, sys.TotalMemMB = memory()
	return sys, procs
}

type cimRow struct {
	pid  int
	ppid int
	wsKB float64
}

func cimRows() []cimRow {
	out, err := runHidden("powershell", "-NoProfile", "-Command",
		"Get-CimInstance Win32_Process | ForEach-Object { \"$($_.ProcessId) $($_.ParentProcessId) $($_.WorkingSetSize)\" }")
	if err != nil {
		return nil
	}
	var rows []cimRow
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(strings.TrimSpace(line))
		if len(f) != 3 {
			continue
		}
		pid, err1 := strconv.Atoi(f[0])
		ppid, err2 := strconv.Atoi(f[1])
		ws, err3 := strconv.ParseFloat(f[2], 64)
		if err1 != nil || err2 != nil || err3 != nil {
			continue
		}
		rows = append(rows, cimRow{pid: pid, ppid: ppid, wsKB: ws / 1024})
	}
	return rows
}

func cpuLoadPercent() float64 {
	out, err := runHidden("powershell", "-NoProfile", "-Command",
		"(Get-CimInstance Win32_Processor | Measure-Object -Property LoadPercentage -Average).Average")
	if err != nil {
		return 0
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(out), 64)
	if err != nil {
		return 0
	}
	return v
}

func runHidden(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	return string(out), err
}
