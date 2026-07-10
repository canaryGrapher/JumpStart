package deps

import (
	"os"
	"path/filepath"
	"strings"
)

func inspectGoMod(dir string) (*Info, error) {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return nil, err
	}
	info := &Info{
		Manager:        "go modules",
		ManifestFile:   "go.mod",
		InstallCommand: "go mod download",
	}
	cache := goModCache()
	for _, req := range parseGoModRequires(string(data)) {
		status := StatusUnknown
		if cache != "" {
			if _, err := os.Stat(filepath.Join(cache, escapeGoPath(req.path)+"@"+escapeGoPath(req.version))); err == nil {
				status = StatusOK
			} else {
				status = StatusMissing
			}
		}
		kind := "dep"
		if req.indirect {
			kind = "indirect"
		}
		info.Dependencies = append(info.Dependencies, Dependency{
			Name: req.path, Version: req.version, Kind: kind, Status: status,
		})
	}
	return info, nil
}

type goRequire struct {
	path, version string
	indirect      bool
}

// parseGoModRequires handles both single-line and block `require` directives.
func parseGoModRequires(content string) []goRequire {
	var out []goRequire
	inBlock := false
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		switch {
		case line == "require (":
			inBlock = true
			continue
		case inBlock && line == ")":
			inBlock = false
			continue
		}
		fields := line
		if !inBlock {
			if !strings.HasPrefix(line, "require ") {
				continue
			}
			fields = strings.TrimPrefix(line, "require ")
		}
		if fields == "" || strings.HasPrefix(fields, "//") {
			continue
		}
		parts := strings.Fields(fields)
		if len(parts) < 2 {
			continue
		}
		out = append(out, goRequire{
			path:     parts[0],
			version:  parts[1],
			indirect: strings.Contains(fields, "// indirect"),
		})
	}
	return out
}

func goModCache() string {
	if c := os.Getenv("GOMODCACHE"); c != "" {
		return c
	}
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		gopath = filepath.Join(home, "go")
	}
	return filepath.Join(gopath, "pkg", "mod")
}

// escapeGoPath applies the module cache encoding: uppercase letters
// become "!" + lowercase (e.g. github.com/Azure -> github.com/!azure).
func escapeGoPath(p string) string {
	var b strings.Builder
	for _, r := range p {
		if r >= 'A' && r <= 'Z' {
			b.WriteByte('!')
			b.WriteRune(r + ('a' - 'A'))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
