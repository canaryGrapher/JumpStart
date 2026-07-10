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
		info.Dependencies = append(info.Dependencies, Dependency{
			Name:    strings.TrimSpace(name),
			Version: strings.Trim(strings.TrimSpace(rest), `"`),
			Kind:    kind,
			Status:  StatusUnknown, // cargo cache lookup needs the toolchain
		})
	}
	return info, nil
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
			Name: name, Version: version, Kind: "dep", Status: StatusUnknown,
		})
	}
	return info, nil
}
