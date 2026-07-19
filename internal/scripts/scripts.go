// Package scripts auto-discovers one-off commands ("scripts") that a
// project directory exposes: npm scripts, Makefile targets, Go flags,
// Cargo/Poetry/Composer/Rake/Gradle tasks and so on.
//
// These become the buttons rendered on a process card, next to Start/Stop.
package scripts

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Found is one detected script candidate.
type Found struct {
	Name    string `json:"name"`    // button label
	Command string `json:"command"` // shell command to run
	Source  string `json:"source"`  // file it came from, e.g. "package.json"
}

// detector inspects a directory and returns any scripts it recognises.
type detector func(dir string) []Found

// detectors runs in order; results are merged and de-duplicated by command.
var detectors = []detector{
	detectNode,
	detectMakefile,
	detectGo,
	detectPython,
	detectRust,
	detectRuby,
	detectPHP,
	detectJVM,
	detectDotNet,
	detectShellScripts,
}

// Detect scans dir and returns every script it can find, sorted by source
// then name. Long-running commands (dev servers, watch modes) are filtered
// out since those belong in the process command, not a script button.
func Detect(dir string) ([]Found, error) {
	if dir = strings.TrimSpace(dir); dir == "" {
		return []Found{}, nil
	}
	info, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, os.ErrInvalid
	}

	var out []Found
	seen := map[string]bool{}
	for _, d := range detectors {
		for _, f := range d(dir) {
			key := f.Command
			if key == "" || seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, f)
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Source != out[j].Source {
			return out[i].Source < out[j].Source
		}
		return out[i].Name < out[j].Name
	})
	if out == nil {
		out = []Found{}
	}
	return out, nil
}

// longRunning matches script names that start a server or watcher; those
// are process commands rather than one-off scripts.
var longRunning = []string{"dev", "start", "serve", "watch", "preview"}

func isLongRunning(name string) bool {
	n := strings.ToLower(name)
	for _, bad := range longRunning {
		if n == bad || strings.HasPrefix(n, bad+":") || strings.HasPrefix(n, bad+"-") {
			return true
		}
	}
	return false
}

func exists(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}

func readFile(dir, name string) ([]byte, bool) {
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return nil, false
	}
	return data, true
}

// label turns a raw script key into a button label: "db:migrate" -> "Db Migrate".
func label(key string) string {
	s := strings.NewReplacer(":", " ", "_", " ", "-", " ", ".", " ").Replace(key)
	words := strings.Fields(s)
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	if len(words) == 0 {
		return key
	}
	return strings.Join(words, " ")
}
