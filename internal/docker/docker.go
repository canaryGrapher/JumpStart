// Package docker wraps the docker CLI to give the app a small, UI-friendly
// API for a project's Docker Compose stack: detecting compose files, bringing
// the stack up/down, and inspecting containers, images and volumes.
//
// Everything shells out to the local `docker` binary (Compose v2 plugin), so
// the host must have Docker installed for any of these calls to succeed.
package docker

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// composeFiles are the compose manifests we recognise, in priority order.
var composeFiles = []string{
	"docker-compose.yml",
	"docker-compose.yaml",
	"compose.yml",
	"compose.yaml",
}

// Info reports what Docker assets a project directory contains and whether
// the docker CLI is usable on this machine.
type Info struct {
	Available     bool   `json:"available"`     // docker CLI found and responsive
	HasCompose    bool   `json:"hasCompose"`    // a compose file exists in root
	HasDockerfile bool   `json:"hasDockerfile"` // a Dockerfile exists in root
	ComposeFile   string `json:"composeFile"`   // relative name of the compose file, if any
	Project       string `json:"project"`       // derived compose project name
}

// Inspect returns Docker info for a project root without touching the network.
func Inspect(root string) Info {
	info := Info{
		Available:     Available(),
		HasDockerfile: fileExists(filepath.Join(root, "Dockerfile")),
	}
	if cf := composeFile(root); cf != "" {
		info.HasCompose = true
		info.ComposeFile = cf
	}
	info.Project = projectName(root)
	return info
}

// Detected reports whether root contains any Docker marker (compose file or
// Dockerfile). The UI uses this to decide whether to show the Containers tab.
func Detected(root string) bool {
	if composeFile(root) != "" {
		return true
	}
	return fileExists(filepath.Join(root, "Dockerfile"))
}

// Available reports whether the docker CLI is installed and responsive.
func Available() bool {
	if _, err := exec.LookPath("docker"); err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, "docker", "version", "--format", "{{.Server.Version}}").Run() == nil
}

// composeFile returns the relative name of the first recognised compose file
// in root, or "" if none exists.
func composeFile(root string) string {
	for _, name := range composeFiles {
		if fileExists(filepath.Join(root, name)) {
			return name
		}
	}
	return ""
}

// projectName derives the Compose project name Docker would use for root:
// the lower-cased directory base name with unsupported characters stripped.
// This matches Docker's default when COMPOSE_PROJECT_NAME is not set.
func projectName(root string) string {
	base := strings.ToLower(filepath.Base(root))
	var b strings.Builder
	for _, r := range base {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		default:
			// drop everything else (dots, spaces, etc.)
		}
	}
	name := strings.TrimLeft(b.String(), "_-")
	return name
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
