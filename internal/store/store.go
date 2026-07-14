package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"devdeck/internal/model"
)

// Store persists projects to ~/.jumpstart/config.json.
type Store struct {
	mu   sync.Mutex
	path string
}

func New() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".jumpstart")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "config.json")
	migrateLegacyConfig(filepath.Join(home, ".devdeck", "config.json"), path)
	return &Store{path: path}, nil
}

func migrateLegacyConfig(oldPath, newPath string) {
	if _, err := os.Stat(newPath); err == nil {
		return
	}
	data, err := os.ReadFile(oldPath)
	if err != nil {
		return
	}
	_ = os.WriteFile(newPath, data, 0o644)
}

func (s *Store) Load() ([]model.Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return []model.Project{}, nil
	}
	if err != nil {
		return nil, err
	}
	var projects []model.Project
	if err := json.Unmarshal(data, &projects); err != nil {
		return nil, err
	}
	return projects, nil
}

func (s *Store) Save(projects []model.Project) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.MarshalIndent(projects, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
