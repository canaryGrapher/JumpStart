package deps

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// --- PHP (composer) ---

func inspectPHP(dir string) (*Info, error) {
	data, err := os.ReadFile(filepath.Join(dir, "composer.json"))
	if err != nil {
		return nil, err
	}
	var pkg struct {
		Require    map[string]string `json:"require"`
		RequireDev map[string]string `json:"require-dev"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}
	info := &Info{
		Manager:        "composer",
		ManifestFile:   "composer.json",
		InstallCommand: "composer install",
	}
	info.Dependencies = append(
		phpDeps(dir, pkg.Require, "dep"),
		phpDeps(dir, pkg.RequireDev, "dev")...,
	)
	return info, nil
}

func phpDeps(dir string, m map[string]string, kind string) []Dependency {
	out := make([]Dependency, 0, len(m))
	for name, version := range m {
		if name == "php" || strings.HasPrefix(name, "ext-") {
			continue // platform requirements, not packages
		}
		status := StatusMissing
		if exists(dir, filepath.Join("vendor", name)) {
			status = StatusOK
		}
		out = append(out, Dependency{Name: name, Version: version, Kind: kind, Status: status})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// --- Rust (cargo) ---

func inspectCargo(dir string) (*Info, error) {
	data, err := os.ReadFile(filepath.Join(dir, "Cargo.toml"))
	if err != nil {
		return nil, err
	}
	info := &Info{
		Manager:        "cargo",
		ManifestFile:   "Cargo.toml",
		InstallCommand: "cargo fetch",
	}
	section := ""
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "[") {
			section = strings.Trim(line, "[]")
			continue
		}
		kind := ""
		switch section {
		case "dependencies":
			kind = "dep"
		case "dev-dependencies":
			kind = "dev"
		default:
			continue
		}
		name, rest, ok := strings.Cut(line, "=")
		if !ok || strings.HasPrefix(line, "#") {
			continue
		}
		name = strings.TrimSpace(name)
		info.Dependencies = append(info.Dependencies, Dependency{
			Name:    name,
			Version: strings.Trim(strings.TrimSpace(rest), `"`),
			Kind:    kind,
			Status:  cargoStatus(name),
		})
	}
	return info, nil
}

func cargoStatus(name string) string {
	roots := cargoRegistryRoots()
	if len(roots) == 0 {
		return StatusUnknown
	}
	for _, root := range roots {
		matches, _ := filepath.Glob(filepath.Join(root, name+"-*"))
		if len(matches) > 0 {
			return StatusOK
		}
	}
	return StatusMissing
}

func cargoRegistryRoots() []string {
	cargoHome := os.Getenv("CARGO_HOME")
	if cargoHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		cargoHome = filepath.Join(home, ".cargo")
	}
	matches, _ := filepath.Glob(filepath.Join(cargoHome, "registry", "src", "*"))
	var roots []string
	for _, m := range matches {
		if info, err := os.Stat(m); err == nil && info.IsDir() {
			roots = append(roots, m)
		}
	}
	return roots
}

// --- Ruby (bundler) ---

func inspectRuby(dir string) (*Info, error) {
	data, err := os.ReadFile(filepath.Join(dir, "Gemfile"))
	if err != nil {
		return nil, err
	}
	info := &Info{
		Manager:        "bundler",
		ManifestFile:   "Gemfile",
		InstallCommand: "bundle install",
	}
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if !strings.HasPrefix(line, "gem ") {
			continue
		}
		parts := strings.Split(strings.TrimPrefix(line, "gem "), ",")
		name := strings.Trim(strings.TrimSpace(parts[0]), `"'`)
		version := ""
		if len(parts) > 1 {
			v := strings.TrimSpace(parts[1])
			if strings.HasPrefix(v, `"`) || strings.HasPrefix(v, `'`) {
				version = strings.Trim(v, `"'`)
			}
		}
		info.Dependencies = append(info.Dependencies, Dependency{
			Name: name, Version: version, Kind: "dep", Status: rubyStatus(dir, name),
		})
	}
	return info, nil
}

func rubyStatus(dir, name string) string {
	roots := rubyGemRoots(dir)
	if len(roots) == 0 {
		return StatusUnknown
	}
	for _, root := range roots {
		matches, _ := filepath.Glob(filepath.Join(root, name+"-*"))
		if len(matches) > 0 {
			return StatusOK
		}
	}
	return StatusMissing
}

func rubyGemRoots(dir string) []string {
	var roots []string
	local, _ := filepath.Glob(filepath.Join(dir, "vendor", "bundle", "ruby", "*", "gems"))
	roots = appendExistingDirs(roots, local)

	home, err := os.UserHomeDir()
	if err == nil {
		user, _ := filepath.Glob(filepath.Join(home, ".gem", "ruby", "*", "gems"))
		roots = appendExistingDirs(roots, user)
	}
	return roots
}

func appendExistingDirs(out, candidates []string) []string {
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			out = append(out, c)
		}
	}
	return out
}
