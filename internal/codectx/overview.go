package codectx

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"devdeck/internal/deps"
	"devdeck/internal/detect"
	"devdeck/internal/scripts"
)

const (
	maxReadmeChars = 4000
	maxTreeLines   = 120
	maxPackages    = 400
)

// buildOverview assembles the cheap summary that is included in every
// prompt: languages, frameworks, dependency manifests, run scripts, a
// condensed tree, and the README.
func buildOverview(name, root string, files []FileEntry) Overview {
	ov := Overview{Name: name, Root: root}
	ov.Languages = countLanguages(files)
	ov.Tree = condensedTree(files)
	ov.Readme = readReadme(root)
	ov.EnvKeys = envKeys(root)

	fwSeen := map[string]bool{}
	dirSeen := map[string]bool{}

	if found, err := detect.Scan(root); err == nil {
		for _, d := range found {
			if d.Framework != "" && !fwSeen[d.Framework] {
				fwSeen[d.Framework] = true
				ov.Frameworks = append(ov.Frameworks, d.Framework)
			}
			if d.Dir != "" && !dirSeen[d.Dir] {
				dirSeen[d.Dir] = true
			}
		}
	}
	sort.Strings(ov.Frameworks)

	// Always inspect the root; add any subfolder that detect flagged.
	dirs := []string{root}
	for d := range dirSeen {
		if d != root {
			dirs = append(dirs, d)
		}
	}
	sort.Strings(dirs)

	for _, dir := range dirs {
		if m, ok := manifestFor(root, dir); ok {
			ov.Manifests = append(ov.Manifests, m)
		}
		ov.Scripts = append(ov.Scripts, scriptsFor(root, dir)...)
	}
	return ov
}

func manifestFor(root, dir string) (Manifest, bool) {
	info, err := deps.Inspect(dir)
	if err != nil || info == nil || info.Manager == "none" {
		return Manifest{}, false
	}
	m := Manifest{
		Dir:       relOrDot(root, dir),
		Manager:   info.Manager,
		File:      info.ManifestFile,
		Install:   info.InstallCommand,
		Installed: info.Installed,
		Missing:   info.Missing,
	}
	for _, d := range info.Dependencies {
		if d.Kind == "indirect" {
			continue // transitive noise; the direct list answers "is X installed"
		}
		if len(m.Packages) >= maxPackages {
			break
		}
		entry := d.Name
		if d.Version != "" {
			entry += "@" + d.Version
		}
		if d.Status == deps.StatusMissing {
			entry += " (not installed)"
		}
		m.Packages = append(m.Packages, entry)
	}
	return m, true
}

func scriptsFor(root, dir string) []RunScript {
	found, err := scripts.Detect(dir)
	if err != nil {
		return nil
	}
	out := make([]RunScript, 0, len(found))
	rel := relOrDot(root, dir)
	for _, f := range found {
		out = append(out, RunScript{
			Dir: rel, Name: f.Name, Command: f.Command, Source: f.Source,
		})
	}
	return out
}

func countLanguages(files []FileEntry) []LangCount {
	tally := map[string]int{}
	for _, f := range files {
		tally[f.Lang]++
	}
	out := make([]LangCount, 0, len(tally))
	for lang, n := range tally {
		out = append(out, LangCount{Lang: lang, Files: n})
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Files > out[b].Files })
	return out
}

// condensedTree lists directories with their file counts rather than every
// path, so a 3000-file repo still fits in a prompt.
func condensedTree(files []FileEntry) []string {
	tally := map[string]int{}
	for _, f := range files {
		dir := path2dir(f.Path)
		tally[dir]++
	}
	dirs := make([]string, 0, len(tally))
	for d := range tally {
		dirs = append(dirs, d)
	}
	sort.Slice(dirs, func(a, b int) bool { return tally[dirs[a]] > tally[dirs[b]] })
	if len(dirs) > maxTreeLines {
		dirs = dirs[:maxTreeLines]
	}
	sort.Strings(dirs)

	out := make([]string, 0, len(dirs))
	for _, d := range dirs {
		out = append(out, d+"/ ("+itoa(tally[d])+" files)")
	}
	return out
}

func path2dir(p string) string {
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[:i]
	}
	return "."
}

func readReadme(root string) string {
	for _, name := range []string{"README.md", "readme.md", "README.MD", "Readme.md"} {
		if data, err := os.ReadFile(filepath.Join(root, name)); err == nil {
			s := string(data)
			if len(s) > maxReadmeChars {
				s = s[:maxReadmeChars] + "\n…(truncated)"
			}
			return s
		}
	}
	return ""
}

// envKeys collects variable names from .env files. Values are never read
// into the index: only the key names, so the model can say which config a
// project expects without leaking secrets.
func envKeys(root string) []string {
	seen := map[string]bool{}
	var out []string

	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != root && (skipDirs[d.Name()] || (strings.HasPrefix(d.Name(), ".") && d.Name() != ".github")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasPrefix(d.Name(), ".env") {
			return nil
		}
		data, readErr := os.ReadFile(p)
		if readErr != nil {
			return nil
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			key, _, ok := strings.Cut(line, "=")
			key = strings.TrimSpace(strings.TrimPrefix(key, "export "))
			if ok && key != "" && !seen[key] {
				seen[key] = true
				out = append(out, key)
			}
		}
		return nil
	})

	sort.Strings(out)
	return out
}

func relOrDot(root, dir string) string {
	rel, err := filepath.Rel(root, dir)
	if err != nil || rel == "" {
		return "."
	}
	return filepath.ToSlash(rel)
}
