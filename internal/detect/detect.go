// Package detect auto-discovers runnable subprocesses inside a project root.
package detect

import (
	"os"
	"path/filepath"
	"sort"
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

// Scan inspects root and its immediate subfolders and returns
// every folder that looks like a runnable process.
func Scan(root string) ([]Detected, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, os.ErrInvalid
	}

	var results []Detected

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() || skipDirs[e.Name()] || e.Name()[0] == '.' {
			continue
		}
		dir := filepath.Join(root, e.Name())
		if d := inspectDir(dir); d != nil {
			results = append(results, *d)
		}
	}

	// Include the root itself if it is runnable (e.g. single-app repo,
	// or a monorepo whose root also has a docker-compose / package.json).
	if d := inspectDir(root); d != nil {
		results = append(results, *d)
	}

	sort.Slice(results, func(i, j int) bool { return results[i].Name < results[j].Name })
	return results, nil
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
