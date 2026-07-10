// Package sysinfo reports CPU/memory usage for process groups
// and for the whole system (macOS and Linux).
package sysinfo

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// ProcUsage is CPU/memory for one managed process group.
type ProcUsage struct {
	CPU   float64 `json:"cpu"`   // percent (100 = one core)
	MemMB float64 `json:"memMB"` // resident memory
}

// SystemUsage is a whole-machine snapshot.
type SystemUsage struct {
	CPU        float64 `json:"cpu"` // percent of total capacity
	UsedMemMB  float64 `json:"usedMemMB"`
	TotalMemMB float64 `json:"totalMemMB"`
	NumCPU     int     `json:"numCpu"`
}

// Snapshot returns system usage plus per-group usage for the given
// process group IDs, using a single `ps` scan.
func Snapshot(pgids map[string]int) (SystemUsage, map[string]ProcUsage) {
	rows := psRows()

	var totalCPU float64
	byPgid := map[int]*ProcUsage{}
	for _, r := range rows {
		totalCPU += r.cpu
		u, ok := byPgid[r.pgid]
		if !ok {
			u = &ProcUsage{}
			byPgid[r.pgid] = u
		}
		u.CPU += r.cpu
		u.MemMB += r.rssKB / 1024
	}

	procs := map[string]ProcUsage{}
	for id, pgid := range pgids {
		if u, ok := byPgid[pgid]; ok {
			procs[id] = *u
		}
	}

	sys := SystemUsage{NumCPU: runtime.NumCPU()}
	if sys.NumCPU > 0 {
		sys.CPU = totalCPU / float64(sys.NumCPU)
	}
	if sys.CPU > 100 {
		sys.CPU = 100
	}
	sys.UsedMemMB, sys.TotalMemMB = memory()
	return sys, procs
}

type psRow struct {
	pgid  int
	cpu   float64
	rssKB float64
}

func psRows() []psRow {
	out, err := exec.Command("ps", "-ax", "-o", "pgid=,pcpu=,rss=").Output()
	if err != nil {
		return nil
	}
	var rows []psRow
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) < 3 {
			continue
		}
		pgid, err1 := strconv.Atoi(f[0])
		cpu, err2 := strconv.ParseFloat(f[1], 64)
		rss, err3 := strconv.ParseFloat(f[2], 64)
		if err1 != nil || err2 != nil || err3 != nil {
			continue
		}
		rows = append(rows, psRow{pgid: pgid, cpu: cpu, rssKB: rss})
	}
	return rows
}
