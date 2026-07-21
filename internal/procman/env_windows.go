//go:build windows

package procman

import "os"

// On Windows, GUI processes inherit the full user PATH, so no fix-up is needed.
func resolvedEnv() []string {
	return os.Environ()
}
