// Package detect auto-discovers runnable subprocesses inside a project root.
package detect

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Detected is a suggested subprocess found by scanning a folder.
type Detected struct {
	Name      string   `json:"name"`
	Dir       string   `json:"dir"`
	Command   string   `json:"command"`
	Language  string   `json:"language"`
	Framework string   `json:"framework"`
	EnvFiles  []string `json:"envFiles"` // file names relative to Dir, e.g. ".env.local"
}

// skipDirs are never scanned as subprocess candidates.
var skipDirs = map[string]bool{
	"node_modules": true, ".git": true, "dist": true, "build": true,
	"vendor": true, "target": true, "__pycache__": true, ".next": true,
	".nuxt": true, ".svelte-kit": true, "venv": true, ".venv": true,
	".idea": true, ".vscode": true, "coverage": true, "out": true,
}

const maxScanDepth = 6

// Scan inspects root recursively and returns every folder that looks like a
// runnable process. Heavy generated/dependency folders are skipped so large
// monorepos remain responsive.
func Scan(root string) ([]Detected, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, os.ErrInvalid
	}

	var results []Detected
	seen := map[string]bool{}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if !entry.IsDir() {
			return nil
		}
		if path != root {
			name := entry.Name()
			if skipDirs[name] || name[0] == '.' {
				return filepath.SkipDir
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return nil
			}
			if depth(rel) > maxScanDepth {
				return filepath.SkipDir
			}
		}
		if d := inspectDir(path); d != nil && !seen[d.Dir] {
			seen[d.Dir] = true
			results = append(results, *d)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(results, func(i, j int) bool { return results[i].Name < results[j].Name })
	return results, nil
}

func depth(rel string) int {
	if rel == "." || rel == "" {
		return 0
	}
	return strings.Count(rel, string(filepath.Separator)) + 1
}

// inspectDir returns a Detected if dir contains a known project marker.
func inspectDir(dir string) *Detected {
	d := detectFramework(dir)
	if d == nil {
		return nil
	}
	d.Dir = dir
	d.EnvFiles = findEnvFiles(dir)
	base := filepath.Base(dir)
	if d.Framework != "" {
		d.Name = base + " (" + d.Framework + ")"
	} else {
		d.Name = base + " (" + d.Language + ")"
	}
	return d
}
