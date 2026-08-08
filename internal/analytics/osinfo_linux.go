package analytics

import (
	"os"
	"strings"
)

// readOSVersion returns the distro ID and version from /etc/os-release,
// e.g. "ubuntu 24.04". Reading the file avoids a subprocess, and the two
// fields together are what a drop-support decision actually needs.
func readOSVersion() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "unknown"
	}
	fields := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		fields[key] = strings.Trim(value, `"'`)
	}
	id, version := fields["ID"], fields["VERSION_ID"]
	switch {
	case id != "" && version != "":
		return id + " " + version
	case id != "":
		return id
	}
	return "unknown"
}
