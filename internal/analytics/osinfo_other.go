//go:build !darwin && !linux && !windows

package analytics

// readOSVersion has no cheap, dependency-free implementation on the
// remaining platforms. "unknown" is an honest answer and keeps the
// property present so breakdowns do not silently drop these installs.
func readOSVersion() string { return "unknown" }
