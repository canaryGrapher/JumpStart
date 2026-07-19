// Package sysinfo reports CPU/memory usage for managed process groups and
// for the whole system. Per-platform collection lives in usage_unix.go
// (ps-based, macOS/Linux) and usage_windows.go (CIM-based).
package sysinfo

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
// process group IDs (root PIDs on Windows).
func Snapshot(pgids map[string]int) (SystemUsage, map[string]ProcUsage) {
	return snapshot(pgids)
}
