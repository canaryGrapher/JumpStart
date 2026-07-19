package scripts

import (
	"encoding/json"
	"strings"
)

type packageJSON struct {
	Scripts        map[string]string `json:"scripts"`
	PackageManager string            `json:"packageManager"`
}

// nodeRunner picks the package manager based on the lockfile present.
func nodeRunner(dir string, pkg packageJSON) string {
	if pm := strings.TrimSpace(pkg.PackageManager); pm != "" {
		if i := strings.Index(pm, "@"); i > 0 {
			return pm[:i]
		}
		return pm
	}
	switch {
	case exists(dir, "pnpm-lock.yaml"):
		return "pnpm"
	case exists(dir, "yarn.lock"):
		return "yarn"
	case exists(dir, "bun.lockb"), exists(dir, "bun.lock"):
		return "bun"
	default:
		return "npm"
	}
}

// nodePlaceholder is what `npm init` writes for an unconfigured test script.
const nodePlaceholder = `echo "Error: no test specified" && exit 1`

func detectNode(dir string) []Found {
	data, ok := readFile(dir, "package.json")
	if !ok {
		return nil
	}
	var pkg packageJSON
	if json.Unmarshal(data, &pkg) != nil || len(pkg.Scripts) == 0 {
		return nil
	}

	runner := nodeRunner(dir, pkg)
	prefix := runner + " run "
	if runner == "yarn" {
		prefix = "yarn "
	}

	var out []Found
	for name, body := range pkg.Scripts {
		if isLongRunning(name) || strings.TrimSpace(body) == nodePlaceholder {
			continue
		}
		out = append(out, Found{
			Name:    label(name),
			Command: prefix + name,
			Source:  "package.json",
		})
	}
	return out
}
