package scripts

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// goFlag matches flag registrations in Go source. Both shapes are
// covered, and the optional pointer argument distinguishes them:
//
//	flag.Bool("migrate", ...)   flag.StringVar(&x, "seed", ...)
//
// Calls without a string literal (flag.Parse) simply never match.
var goFlag = regexp.MustCompile(`flag\.[A-Za-z]+\(\s*(?:&[A-Za-z0-9_.]+\s*,\s*)?"([A-Za-z0-9][A-Za-z0-9_-]*)"`)

// goFlagIgnore are flags that configure a run rather than trigger a task.
var goFlagIgnore = map[string]bool{
	"port": true, "host": true, "addr": true, "config": true,
	"verbose": true, "debug": true, "env": true, "help": true, "v": true,
}

func detectGo(dir string) []Found {
	if !exists(dir, "go.mod") {
		return nil
	}

	out := []Found{
		{Name: "Build", Command: "go build ./...", Source: "go.mod"},
		{Name: "Generate", Command: "go generate ./...", Source: "go.mod"},
		{Name: "Tidy", Command: "go mod tidy", Source: "go.mod"},
		{Name: "Vet", Command: "go vet ./...", Source: "go.mod"},
	}
	for _, flag := range goEntrypointFlags(dir) {
		out = append(out, Found{
			Name:    label(flag),
			Command: "go run . --" + flag,
			Source:  "go.mod",
		})
	}
	return out
}

// goEntrypointFlags scans top-level .go files for CLI flags that look like
// task triggers (migrate, seed, rollback…), so they can become buttons.
func goEntrypointFlags(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var flags []string
	seen := map[string]bool{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		for _, m := range goFlag.FindAllStringSubmatch(string(data), -1) {
			name := m[1]
			if goFlagIgnore[strings.ToLower(name)] || seen[name] {
				continue
			}
			seen[name] = true
			flags = append(flags, name)
		}
	}
	return flags
}
