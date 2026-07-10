package deps

import (
	"os"
	"path/filepath"
	"strings"
)

func inspectPython(dir string) (*Info, error) {
	data, err := os.ReadFile(filepath.Join(dir, "requirements.txt"))
	if err != nil {
		return nil, err
	}
	info := &Info{
		Manager:        "pip",
		ManifestFile:   "requirements.txt",
		InstallCommand: pipInstallCommand(dir),
	}
	site := sitePackagesDir(dir)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}
		name, version := splitPythonReq(line)
		if name == "" {
			continue
		}
		status := StatusUnknown
		if site != "" {
			if pythonPkgInstalled(site, name) {
				status = StatusOK
			} else {
				status = StatusMissing
			}
		}
		info.Dependencies = append(info.Dependencies, Dependency{
			Name: name, Version: version, Kind: "dep", Status: status,
		})
	}
	return info, nil
}

// splitPythonReq turns "requests[socks]>=2.0,<3" into ("requests", ">=2.0,<3").
func splitPythonReq(line string) (name, version string) {
	if i := strings.Index(line, ";"); i >= 0 { // env markers
		line = strings.TrimSpace(line[:i])
	}
	// drop extras: "uvicorn[standard]>=0.23" -> "uvicorn>=0.23"
	if open := strings.Index(line, "["); open >= 0 {
		if close := strings.Index(line, "]"); close > open {
			line = line[:open] + line[close+1:]
		} else {
			line = line[:open]
		}
	}
	for i, r := range line {
		if strings.ContainsRune("=<>!~ ", r) {
			return strings.TrimSpace(line[:i]), strings.TrimSpace(line[i:])
		}
	}
	return line, ""
}

// sitePackagesDir finds the venv site-packages folder for dir, if any.
func sitePackagesDir(dir string) string {
	for _, venv := range []string{".venv", "venv", "env"} {
		matches, _ := filepath.Glob(filepath.Join(dir, venv, "lib", "python*", "site-packages"))
		if len(matches) > 0 {
			return matches[0]
		}
		// Windows layout
		win := filepath.Join(dir, venv, "Lib", "site-packages")
		if _, err := os.Stat(win); err == nil {
			return win
		}
	}
	return ""
}

// pythonPkgInstalled checks for the package folder or its dist-info.
func pythonPkgInstalled(site, name string) bool {
	norm := strings.ToLower(strings.ReplaceAll(name, "-", "_"))
	if _, err := os.Stat(filepath.Join(site, norm)); err == nil {
		return true
	}
	matches, _ := filepath.Glob(filepath.Join(site, norm+"-*.dist-info"))
	return len(matches) > 0
}

func pipInstallCommand(dir string) string {
	for _, venv := range []string{".venv", "venv", "env"} {
		if exists(dir, filepath.Join(venv, "bin", "pip")) {
			return venv + "/bin/pip install -r requirements.txt"
		}
		if exists(dir, filepath.Join(venv, "Scripts", "pip.exe")) {
			return venv + "\\Scripts\\pip install -r requirements.txt"
		}
	}
	return "pip install -r requirements.txt"
}
