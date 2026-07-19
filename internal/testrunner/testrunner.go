// Package testrunner detects the test command for a project directory
// (Node, Go, Python) so the app can offer a one-click "Run tests" action.
package testrunner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// TestConfig describes how to run tests for a project directory.
type TestConfig struct {
	Detected bool     `json:"detected"`
	Kind     string   `json:"kind"` // "node" | "go" | "python" | "none"
	Command  string   `json:"command"`
	Args     []string `json:"args"`
	Dir      string   `json:"dir"`
}

func exists(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}

// npmDefaultTestScript is the placeholder `npm init` writes when no test
// runner has been configured; a project with only this script has no
// real tests to run.
const npmDefaultTestScript = `echo "Error: no test specified" && exit 1`

// Detect inspects dir and returns the test command to run, if any.
func Detect(dir string) (*TestConfig, error) {
	if exists(dir, "package.json") {
		if cfg := detectNode(dir); cfg != nil {
			return cfg, nil
		}
	}
	if exists(dir, "go.mod") {
		return &TestConfig{
			Detected: true,
			Kind:     "go",
			Command:  "go test ./...",
			Dir:      dir,
		}, nil
	}
	if cfg := detectPython(dir); cfg != nil {
		return cfg, nil
	}
	return &TestConfig{Detected: false, Kind: "none", Dir: dir}, nil
}

type packageJSON struct {
	Scripts map[string]string `json:"scripts"`
}

func detectNode(dir string) *TestConfig {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return nil
	}
	var pkg packageJSON
	if json.Unmarshal(data, &pkg) != nil {
		return nil
	}
	script, ok := pkg.Scripts["test"]
	if !ok || strings.TrimSpace(script) == "" || strings.TrimSpace(script) == npmDefaultTestScript {
		return nil
	}
	return &TestConfig{
		Detected: true,
		Kind:     "node",
		Command:  nodeTestCommand(dir),
		Dir:      dir,
	}
}

// nodeTestCommand mirrors detect.nodePackageManager's lockfile-based
// package-manager detection to pick the right "test" runner invocation.
func nodeTestCommand(dir string) string {
	switch {
	case exists(dir, "pnpm-lock.yaml"):
		return "pnpm test"
	case exists(dir, "yarn.lock"):
		return "yarn test"
	case exists(dir, "bun.lockb"), exists(dir, "bun.lock"):
		return "bun run test"
	default:
		return "npm test"
	}
}

func detectPython(dir string) *TestConfig {
	hasPytestConfig := exists(dir, "pytest.ini")
	if !hasPytestConfig && exists(dir, "pyproject.toml") {
		if data, err := os.ReadFile(filepath.Join(dir, "pyproject.toml")); err == nil {
			if strings.Contains(string(data), "[tool.pytest") {
				hasPytestConfig = true
			}
		}
	}
	hasPytestDep := false
	if data, err := os.ReadFile(filepath.Join(dir, "requirements.txt")); err == nil {
		if strings.Contains(strings.ToLower(string(data)), "pytest") {
			hasPytestDep = true
		}
	}
	hasTestsDir := false
	if entries, err := os.ReadDir(filepath.Join(dir, "tests")); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".py") {
				hasTestsDir = true
				break
			}
		}
	}

	if hasPytestConfig || hasPytestDep || hasTestsDir {
		return &TestConfig{
			Detected: true,
			Kind:     "python",
			Command:  "pytest",
			Dir:      dir,
		}
	}
	return nil
}
