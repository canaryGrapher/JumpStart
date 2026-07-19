//go:build windows

package sysinfo

import (
	"strconv"
	"strings"
)

// memory returns (usedMB, totalMB) from Win32_OperatingSystem, which
// reports sizes in kilobytes.
func memory() (used, total float64) {
	out, err := runHidden("powershell", "-NoProfile", "-Command",
		"$os = Get-CimInstance Win32_OperatingSystem; \"$($os.TotalVisibleMemorySize) $($os.FreePhysicalMemory)\"")
	if err != nil {
		return 0, 0
	}
	f := strings.Fields(strings.TrimSpace(out))
	if len(f) != 2 {
		return 0, 0
	}
	totalKB, err1 := strconv.ParseFloat(f[0], 64)
	freeKB, err2 := strconv.ParseFloat(f[1], 64)
	if err1 != nil || err2 != nil {
		return 0, 0
	}
	return (totalKB - freeKB) / 1024, totalKB / 1024
}
