// Package config imports projects from a user-editable conf file,
// so projects can be added programmatically (e.g. by an AI agent).
package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"devdeck/internal/model"
)

// ImportPath is where the conf file is expected: ~/.devdeck/import.json
func ImportPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".devdeck", "import.json"), nil
}

// confFile allows either a bare array of projects or a wrapper object.
type confFile struct {
	Projects []model.Project `json:"projects"`
}

// Load reads and validates the conf file, filling in missing IDs.
func Load() ([]model.Project, error) {
	path, err := ImportPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("no conf file found at %s", path)
	}
	if err != nil {
		return nil, err
	}

	var wrapped confFile
	if err := json.Unmarshal(data, &wrapped); err != nil || wrapped.Projects == nil {
		var bare []model.Project
		if err2 := json.Unmarshal(data, &bare); err2 != nil {
			return nil, fmt.Errorf("invalid conf file: %v", err)
		}
		wrapped.Projects = bare
	}

	for i := range wrapped.Projects {
		p := &wrapped.Projects[i]
		if p.Name == "" {
			return nil, fmt.Errorf("project %d has no name", i+1)
		}
		if p.ID == "" {
			p.ID = newID()
		}
		for j := range p.Processes {
			if p.Processes[j].ID == "" {
				p.Processes[j].ID = newID()
			}
		}
		for j := range p.Tasks {
			if p.Tasks[j].ID == "" {
				p.Tasks[j].ID = newID()
			}
		}
	}
	return wrapped.Projects, nil
}

// Merge upserts imported projects into existing ones.
// Match is by ID first, then by name. Returns (added, updated).
func Merge(existing, imported []model.Project) ([]model.Project, int, int) {
	added, updated := 0, 0
	for _, imp := range imported {
		idx := -1
		for i, ex := range existing {
			if ex.ID == imp.ID || ex.Name == imp.Name {
				idx = i
				break
			}
		}
		if idx == -1 {
			existing = append(existing, imp)
			added++
			continue
		}
		// keep usage stats and identity of the existing project
		imp.ID = existing[idx].ID
		if imp.LastUsedAt == 0 {
			imp.LastUsedAt = existing[idx].LastUsedAt
		}
		if imp.UseCount == 0 {
			imp.UseCount = existing[idx].UseCount
		}
		existing[idx] = imp
		updated++
	}
	return existing, added, updated
}

func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "id-fallback"
	}
	// format as UUID-like for consistency with crypto.randomUUID()
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(b[0:4]), hex.EncodeToString(b[4:6]),
		hex.EncodeToString(b[6:8]), hex.EncodeToString(b[8:10]),
		hex.EncodeToString(b[10:16]))
}
