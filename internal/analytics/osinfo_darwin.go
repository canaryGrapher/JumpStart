package analytics

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// readOSVersion returns the macOS product version, e.g. "15.3". The
// subprocess is bounded so a wedged sw_vers cannot delay app startup.
func readOSVersion() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "sw_vers", "-productVersion").Output()
	if err != nil {
		return "unknown"
	}
	v := strings.TrimSpace(string(out))
	if v == "" {
		return "unknown"
	}
	return v
}
