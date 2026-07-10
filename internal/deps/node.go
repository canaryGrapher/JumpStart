package deps

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

func inspectNode(dir string) (*Info, error) {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return nil, err
	}
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}

	manager, installCmd := nodeManager(dir)
	info := &Info{
		Manager:        manager,
		ManifestFile:   "package.json",
		InstallCommand: installCmd,
	}
	info.Dependencies = append(
		nodeDeps(dir, pkg.Dependencies, "dep"),
		nodeDeps(dir, pkg.DevDependencies, "dev")...,
	)
	return info, nil
}

func nodeDeps(dir string, m map[string]string, kind string) []Dependency {
	out := make([]Dependency, 0, len(m))
	for name, version := range m {
		status := StatusMissing
		if exists(dir, filepath.Join("node_modules", name, "package.json")) {
			status = StatusOK
		}
		out = append(out, Dependency{Name: name, Version: version, Kind: kind, Status: status})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func nodeManager(dir string) (manager, installCmd string) {
	switch {
	case exists(dir, "pnpm-lock.yaml"):
		return "pnpm", "pnpm install"
	case exists(dir, "yarn.lock"):
		return "yarn", "yarn install"
	case exists(dir, "bun.lockb"), exists(dir, "bun.lock"):
		return "bun", "bun install"
	default:
		return "npm", "npm install"
	}
}
