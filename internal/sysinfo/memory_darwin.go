//go:build darwin

package sysinfo

import (
	"os/exec"
	"strconv"
	"strings"
)

// memory returns (usedMB, totalMB) on macOS via sysctl + vm_stat.
func memory() (used, total float64) {
	if out, err := exec.Command("sysctl", "-n", "hw.memsize").Output(); err == nil {
		if b, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64); err == nil {
			total = b / 1024 / 1024
		}
	}
	out, err := exec.Command("vm_stat").Output()
	if err != nil {
		return 0, total
	}
	pageSize := 16384.0
	var activeP, wiredP, compP float64
	for _, line := range strings.Split(string(out), "\n") {
		switch {
		case strings.HasPrefix(line, "Mach Virtual Memory Statistics"):
			if i := strings.Index(line, "page size of "); i >= 0 {
				f := strings.Fields(line[i+len("page size of "):])
				if len(f) > 0 {
					if v, err := strconv.ParseFloat(f[0], 64); err == nil {
						pageSize = v
					}
				}
			}
		case strings.HasPrefix(line, "Pages active:"):
			activeP = vmStatValue(line)
		case strings.HasPrefix(line, "Pages wired down:"):
			wiredP = vmStatValue(line)
		case strings.HasPrefix(line, "Pages occupied by compressor:"):
			compP = vmStatValue(line)
		}
	}
	used = (activeP + wiredP + compP) * pageSize / 1024 / 1024
	return used, total
}

func vmStatValue(line string) float64 {
	f := strings.Fields(line)
	if len(f) == 0 {
		return 0
	}
	v, _ := strconv.ParseFloat(strings.TrimSuffix(f[len(f)-1], "."), 64)
	return v
}
