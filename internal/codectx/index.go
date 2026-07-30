package codectx

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Progress reports indexing progress to the caller (used to emit Wails
// events so the UI can show a live counter).
type Progress func(done, total int, path string)

// Build scans root, chunks every text file, and returns a ready index.
// The result is not written to disk; call Save for that.
func Build(projectID, name, root string, onProgress Progress) (*Index, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("project has no root folder")
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a folder", root)
	}

	files, skipped := walk(root)

	ix := &Index{
		ProjectID: projectID,
		Root:      root,
		BuiltAt:   time.Now().UnixMilli(),
		Skipped:   skipped,
		Files:     make([]FileEntry, 0, len(files)),
	}

	for i, f := range files {
		text, ok := readText(root, f.Path)
		if !ok {
			ix.Skipped++
			continue
		}
		f.Lines = strings.Count(text, "\n") + 1
		ix.Files = append(ix.Files, f)
		ix.Chunks = append(ix.Chunks, chunkFile(f.Path, f.Lang, text)...)
		if onProgress != nil && i%25 == 0 {
			onProgress(i, len(files), f.Path)
		}
	}

	ix.Overview = buildOverview(name, root, ix.Files)
	ix.prepare()
	if onProgress != nil {
		onProgress(len(files), len(files), "")
	}
	return ix, nil
}

// --- persistence -------------------------------------------------------

var saveMu sync.Mutex

func indexDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".jumpstart", "context")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func indexPath(projectID string) (string, error) {
	dir, err := indexDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, safeName(projectID)+".json"), nil
}

// safeName keeps a project ID usable as a filename.
func safeName(id string) string {
	var b strings.Builder
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	if b.Len() == 0 {
		return "project"
	}
	return b.String()
}

// Save writes the index atomically.
func Save(ix *Index) error {
	saveMu.Lock()
	defer saveMu.Unlock()

	path, err := indexPath(ix.ProjectID)
	if err != nil {
		return err
	}
	data, err := json.Marshal(ix)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Load reads a previously built index. Returns nil (no error) when the
// project has never been indexed.
func Load(projectID string) (*Index, error) {
	path, err := indexPath(projectID)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ix Index
	if err := json.Unmarshal(data, &ix); err != nil {
		return nil, err
	}
	ix.prepare()
	return &ix, nil
}

// Delete removes a project's index from disk.
func Delete(projectID string) error {
	path, err := indexPath(projectID)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// --- in-memory cache ---------------------------------------------------

var (
	cacheMu sync.RWMutex
	cache   = map[string]*Index{}
)

// Get returns a project's index, loading and caching it on first use.
func Get(projectID string) (*Index, error) {
	cacheMu.RLock()
	ix, ok := cache[projectID]
	cacheMu.RUnlock()
	if ok {
		return ix, nil
	}

	loaded, err := Load(projectID)
	if err != nil || loaded == nil {
		return nil, err
	}
	cacheMu.Lock()
	cache[projectID] = loaded
	cacheMu.Unlock()
	return loaded, nil
}

// Put caches an index (called after a rebuild).
func Put(ix *Index) {
	cacheMu.Lock()
	cache[ix.ProjectID] = ix
	cacheMu.Unlock()
}

// Evict drops a project from the cache.
func Evict(projectID string) {
	cacheMu.Lock()
	delete(cache, projectID)
	cacheMu.Unlock()
}
