package codectx

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Limits keep indexing responsive on large monorepos.
const (
	maxFileBytes = 256 * 1024 // skip anything bigger; generated bundles mostly
	maxFiles     = 4000
	maxDepth     = 12
)

// skipDirs are never descended into. Mirrors internal/detect plus a few
// extras that only matter when reading file contents.
var skipDirs = map[string]bool{
	"node_modules": true, ".git": true, "dist": true, "build": true,
	"vendor": true, "target": true, "__pycache__": true, ".next": true,
	".nuxt": true, ".svelte-kit": true, "venv": true, ".venv": true,
	".idea": true, ".vscode": true, "coverage": true, "out": true,
	".turbo": true, ".cache": true, ".parcel-cache": true, "bin": true,
	".pnpm": true, ".yarn": true, "Pods": true, "DerivedData": true,
	".terraform": true, ".gradle": true, ".mypy_cache": true,
	".pytest_cache": true, "wailsjs": true,
}

// langByExt maps a file extension to the language label used in prompts.
var langByExt = map[string]string{
	".go": "go", ".js": "javascript", ".jsx": "jsx", ".ts": "typescript",
	".tsx": "tsx", ".py": "python", ".rb": "ruby", ".php": "php",
	".rs": "rust", ".java": "java", ".kt": "kotlin", ".swift": "swift",
	".c": "c", ".h": "c", ".cpp": "cpp", ".hpp": "cpp", ".cs": "csharp",
	".css": "css", ".scss": "scss", ".less": "less", ".html": "html",
	".vue": "vue", ".svelte": "svelte", ".sql": "sql", ".sh": "shell",
	".bash": "shell", ".zsh": "shell", ".yml": "yaml", ".yaml": "yaml",
	".json": "json", ".toml": "toml", ".md": "markdown", ".mdx": "markdown",
	".proto": "proto", ".graphql": "graphql", ".tf": "terraform",
	".dockerfile": "docker",
}

// skipFiles are lockfiles and generated artefacts: huge and low signal.
var skipFiles = map[string]bool{
	"package-lock.json": true, "yarn.lock": true, "pnpm-lock.yaml": true,
	"go.sum": true, "Cargo.lock": true, "composer.lock": true,
	"Gemfile.lock": true, "poetry.lock": true, "bun.lockb": true,
}

// langFor returns the language label for a path, or "" if the file is not
// a text source we want to index.
func langFor(path string) string {
	base := filepath.Base(path)
	if skipFiles[base] {
		return ""
	}
	switch base {
	case "Dockerfile", "Dockerfile.dev", "Containerfile":
		return "docker"
	case "Makefile", "makefile":
		return "make"
	}
	// .env files are never indexed: only their key names are collected, by
	// envKeys in overview.go. Indexing the bodies would put secrets in the
	// index and, from there, into model prompts.
	if strings.HasPrefix(base, ".env") {
		return ""
	}
	return langByExt[strings.ToLower(filepath.Ext(path))]
}

// gitignore is a deliberately simple matcher: it honours plain and
// directory patterns, which covers the vast majority of real .gitignore
// files without pulling in a dependency.
type gitignore struct {
	patterns []string
}

func loadGitignore(root string) *gitignore {
	f, err := os.Open(filepath.Join(root, ".gitignore"))
	if err != nil {
		return &gitignore{}
	}
	defer f.Close()

	var pats []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		pats = append(pats, strings.Trim(line, "/"))
	}
	return &gitignore{patterns: pats}
}

// match reports whether a repo-relative path is ignored.
func (g *gitignore) match(rel string) bool {
	if len(g.patterns) == 0 {
		return false
	}
	base := filepath.Base(rel)
	for _, p := range g.patterns {
		if p == base || p == rel || strings.HasPrefix(rel, p+"/") {
			return true
		}
		if ok, _ := filepath.Match(p, base); ok {
			return true
		}
	}
	return false
}

// walk collects every indexable text file under root.
func walk(root string) (files []FileEntry, skipped int) {
	ig := loadGitignore(root)

	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || len(files) >= maxFiles {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if strings.Count(rel, "/") > maxDepth {
			return filepath.SkipDir
		}

		if entry.IsDir() {
			name := entry.Name()
			if skipDirs[name] || (strings.HasPrefix(name, ".") && name != ".github") {
				return filepath.SkipDir
			}
			if ig.match(rel) {
				return filepath.SkipDir
			}
			return nil
		}

		lang := langFor(path)
		if lang == "" || ig.match(rel) {
			return nil
		}
		info, statErr := entry.Info()
		if statErr != nil {
			return nil
		}
		if info.Size() > maxFileBytes || info.Size() == 0 {
			skipped++
			return nil
		}
		files = append(files, FileEntry{Path: rel, Lang: lang, Size: info.Size()})
		return nil
	})
	return files, skipped
}

// readText loads a file and rejects anything that looks binary.
func readText(root, rel string) (string, bool) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		return "", false
	}
	// A NUL byte in the first KB is a reliable binary signal.
	head := data
	if len(head) > 1024 {
		head = head[:1024]
	}
	for _, b := range head {
		if b == 0 {
			return "", false
		}
	}
	return string(data), true
}
