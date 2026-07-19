//go:build !darwin && !windows

package sysinfo

import (
	"os"
	"strconv"
	"strings"
)

// memory returns (usedMB, totalMB) from /proc/meminfo.
func memory() (used, total float64) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	var totalKB, availKB float64
	for _, line := range strings.Split(string(data), "\n") {
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		v, _ := strconv.ParseFloat(f[1], 64)
		switch f[0] {
		case "MemTotal:":
			totalKB = v
		case "MemAvailable:":
			availKB = v
		}
	}
	return (totalKB - availKB) / 1024, totalKB / 1024
}
