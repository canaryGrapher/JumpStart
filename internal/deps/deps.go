// Package deps lists a folder's dependencies, checks whether they are
// installed, and suggests the command to install them.
package deps

import (
	"os"
	"path/filepath"
)

// Status of a single dependency.
const (
	StatusOK      = "ok"      // installed
	StatusMissing = "missing" // declared but not installed
	StatusUnknown = "unknown" // cannot determine without running the toolchain
)

// Dependency is one entry from a manifest file.
type Dependency struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Kind    string `json:"kind"`   // "dep", "dev" or "indirect"
	Status  string `json:"status"` // see Status* constants
}

// Info summarises the dependencies of one folder.
type Info struct {
	Manager        string       `json:"manager"`        // e.g. "npm", "go modules"
	ManifestFile   string       `json:"manifestFile"`   // e.g. "package.json"
	InstallCommand string       `json:"installCommand"` // e.g. "npm install"
	Installed      int          `json:"installed"`
	Missing        int          `json:"missing"`
	Unknown        int          `json:"unknown"`
	Dependencies   []Dependency `json:"dependencies"`
}

// Inspect detects the package manager used in dir and returns its
// dependencies with their installed state.
func Inspect(dir string) (*Info, error) {
	var info *Info
	var err error
	switch {
	case exists(dir, "package.json"):
		info, err = inspectNode(dir)
	case exists(dir, "go.mod"):
		info, err = inspectGoMod(dir)
	case exists(dir, "requirements.txt"):
		info, err = inspectPython(dir)
	case exists(dir, "composer.json"):
		info, err = inspectPHP(dir)
	case exists(dir, "Cargo.toml"):
		info, err = inspectCargo(dir)
	case exists(dir, "Gemfile"):
		info, err = inspectRuby(dir)
	default:
		return &Info{Manager: "none", Dependencies: []Dependency{}}, nil
	}
	if err != nil {
		return nil, err
	}
	for _, d := range info.Dependencies {
		switch d.Status {
		case StatusOK:
			info.Installed++
		case StatusMissing:
			info.Missing++
		default:
			info.Unknown++
		}
	}
	if info.Dependencies == nil {
		info.Dependencies = []Dependency{}
	}
	return info, nil
}

func exists(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}
