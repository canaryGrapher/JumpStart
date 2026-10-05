package mcpserver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const maxReadBytes = 1 << 20 // 1 MiB

// resolveUnderRoot joins rel to root and ensures the result stays inside root.
func resolveUnderRoot(root, rel string) (string, error) {
	if root == "" {
		return "", fmt.Errorf("project root is empty")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	absRoot = filepath.Clean(absRoot)

	rel = strings.TrimSpace(rel)
	if rel == "" || rel == "." {
		return absRoot, nil
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("path must be relative to the project root")
	}

	joined := filepath.Clean(filepath.Join(absRoot, rel))
	prefix := absRoot + string(os.PathSeparator)
	if joined != absRoot && !strings.HasPrefix(joined, prefix) {
		return "", fmt.Errorf("path escapes project root")
	}
	return joined, nil
}

func readProjectFile(root, rel string) (string, error) {
	path, err := resolveUnderRoot(root, rel)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s is a directory", rel)
	}
	if info.Size() > maxReadBytes {
		return "", fmt.Errorf("file larger than %d bytes", maxReadBytes)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func writeProjectFile(root, rel, content string) error {
	path, err := resolveUnderRoot(root, rel)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".jumpstart-tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

type dirEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"isDir"`
	Size  int64  `json:"size,omitempty"`
}

func listProjectDir(root, rel string) ([]dirEntry, error) {
	path, err := resolveUnderRoot(root, rel)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	out := make([]dirEntry, 0, len(entries))
	base := strings.Trim(filepath.ToSlash(rel), "/")
	for _, e := range entries {
		name := e.Name()
		child := name
		if base != "" && base != "." {
			child = base + "/" + name
		}
		info, _ := e.Info()
		ent := dirEntry{Name: name, Path: child, IsDir: e.IsDir()}
		if info != nil && !e.IsDir() {
			ent.Size = info.Size()
		}
		out = append(out, ent)
	}
	return out, nil
}
