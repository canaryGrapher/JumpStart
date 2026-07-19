//go:build !windows

package sysinfo

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// snapshot uses a single `ps` scan, attributing usage to process groups.
func snapshot(pgids map[string]int) (SystemUsage, map[string]ProcUsage) {
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
