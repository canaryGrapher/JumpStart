package ocr

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"devdeck/internal/attachments"
	"devdeck/internal/model"
)

// Text states, matching search.TextReady etc.
const (
	stateReady   = "ready"
	statePending = "pending"
	stateNone    = "none"
)

type entry struct {
	Engine string `json:"engine"`
	Text   string `json:"text"`
	Error  string `json:"error,omitempty"`
	At     int64  `json:"at"`
}

type job struct {
	key, projectID, taskID string
	att                    model.Attachment
}

type memo struct {
	size, mod int64
	text      string
}

// Worker extracts attachment text in the background and caches it.
type Worker struct {
	dataDir  string
	config   func() Config
	onChange func()

	mu     sync.Mutex
	queued map[string]bool
	jobs   chan job
	texts  sync.Map // path -> memo, for plain text files
	notify *time.Timer
}

// NewWorker starts a worker. config is read for each job so engine changes
// apply immediately; onChange is called (debounced) when new text lands.
func NewWorker(dataDir string, config func() Config, onChange func()) *Worker {
	w := &Worker{dataDir: dataDir, config: config, onChange: onChange, queued: map[string]bool{}, jobs: make(chan job, 1024)}
	go w.loop()
	return w
}

func safe(s string) string {
	s = filepath.Base(filepath.Clean("/" + s))
	if s == "/" || s == "." || s == ".." {
		return "_"
	}
	return s
}

func (w *Worker) root() string { return filepath.Join(w.dataDir, "search-text") }

func (w *Worker) cachePath(projectID, taskID string, a model.Attachment) string {
	return filepath.Join(w.root(), safe(projectID), safe(taskID), safe(a.File)+".json")
}

func (w *Worker) read(path string) (entry, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return entry{}, false
	}
	var e entry
	if json.Unmarshal(data, &e) != nil {
		return entry{}, false
	}
	return e, true
}

// Text implements search.TextSource.
func (w *Worker) Text(projectID, taskID string, a model.Attachment) (string, string) {
	kind := Kind(a)
	if kind == "" {
		return "", stateNone
	}
	path, err := attachments.Path(w.dataDir, projectID, taskID, a)
	if err != nil {
		return "", stateNone
	}
	if kind == "text" {
		st, err := os.Stat(path)
		if err != nil {
			return "", stateNone
		}
		if m, ok := w.texts.Load(path); ok {
			if mm := m.(memo); mm.size == st.Size() && mm.mod == st.ModTime().UnixNano() {
				return mm.text, stateReady
			}
		}
		text, err := ReadText(path)
		if err != nil {
			return "", stateNone
		}
		w.texts.Store(path, memo{size: st.Size(), mod: st.ModTime().UnixNano(), text: text})
		return text, stateReady
	}
	cache := w.cachePath(projectID, taskID, a)
	if e, ok := w.read(cache); ok {
		if e.Text == "" {
			return "", stateNone
		}
		return e.Text, stateReady
	}
	cfg := w.config()
	if kind == "image" && (cfg.Engine == Off || cfg.Engine == "") {
		return "", stateNone
	}
	if kind == "pdf" && findTool("pdftotext") == "" {
		return "", stateNone
	}
	w.enqueue(job{key: cache, projectID: projectID, taskID: taskID, att: a})
	return "", statePending
}

func (w *Worker) enqueue(j job) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.queued[j.key] {
		return
	}
	select {
	case w.jobs <- j:
		w.queued[j.key] = true
	default: // full: it stays pending and is offered again on the next search
	}
}

// Pending is the number of files waiting for extraction.
func (w *Worker) Pending() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.queued)
}

func (w *Worker) loop() {
	for j := range w.jobs {
		w.process(j)
		w.mu.Lock()
		delete(w.queued, j.key)
		w.mu.Unlock()
		w.changed()
	}
}

func (w *Worker) process(j job) {
	path, err := attachments.Path(w.dataDir, j.projectID, j.taskID, j.att)
	if err != nil {
		return
	}
	if _, err := os.Stat(path); err != nil {
		return // removed meanwhile
	}
	cfg := w.config()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	text, err := Extract(ctx, w.dataDir, cfg, path, j.att)
	e := entry{Engine: cfg.Engine, Text: text, At: time.Now().UnixMilli()}
	if Kind(j.att) == "pdf" {
		e.Engine = "pdftotext"
	}
	if err != nil {
		if errors.Is(err, ErrUnsupported) && cfg.Engine == Off {
			return // not cached, so turning OCR on later picks it up
		}
		e.Error = err.Error()
	}
	if err := os.MkdirAll(filepath.Dir(j.key), 0o755); err != nil {
		return
	}
	data, _ := json.Marshal(e)
	tmp := j.key + ".tmp"
	if os.WriteFile(tmp, data, 0o644) == nil {
		_ = os.Rename(tmp, j.key)
	}
}

func (w *Worker) changed() {
	if w.onChange == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.notify != nil {
		w.notify.Stop()
	}
	w.notify = time.AfterFunc(400*time.Millisecond, w.onChange)
}

// Info is the stored extraction result of one attachment.
type Info struct {
	State  string `json:"state"` // ready | pending | none
	Engine string `json:"engine,omitempty"`
	Chars  int    `json:"chars"`
	Error  string `json:"error,omitempty"`
	At     int64  `json:"at,omitempty"`
}

// Status reports what is known about one attachment's text without
// queueing work.
func (w *Worker) Status(projectID, taskID string, a model.Attachment) Info {
	switch Kind(a) {
	case "":
		return Info{State: stateNone}
	case "text":
		text, state := w.Text(projectID, taskID, a)
		return Info{State: state, Engine: "text", Chars: len([]rune(text))}
	}
	cache := w.cachePath(projectID, taskID, a)
	if e, ok := w.read(cache); ok {
		st := stateReady
		if e.Text == "" {
			st = stateNone
		}
		return Info{State: st, Engine: e.Engine, Chars: len([]rune(e.Text)), Error: e.Error, At: e.At}
	}
	w.mu.Lock()
	queued := w.queued[cache]
	w.mu.Unlock()
	if queued {
		return Info{State: statePending}
	}
	return Info{State: stateNone}
}

// Forget drops one attachment's cached text so it is read again.
func (w *Worker) Forget(projectID, taskID string, a model.Attachment) {
	_ = os.Remove(w.cachePath(projectID, taskID, a))
}

// Reset drops cached image text: all of it, or (failedOnly) just entries
// that ended in an error, e.g. after the user fixes the OCR settings.
func (w *Worker) Reset(failedOnly bool) int {
	n := 0
	_ = filepath.WalkDir(w.root(), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		e, ok := w.read(path)
		if !ok || e.Engine == "pdftotext" && !failedOnly {
			return nil
		}
		if failedOnly && e.Error == "" {
			return nil
		}
		if os.Remove(path) == nil {
			n++
		}
		return nil
	})
	return n
}

// Prune deletes cached text for attachments that no longer exist.
func (w *Worker) Prune(projects []model.Project) {
	keep := map[string]bool{}
	for _, p := range projects {
		for _, t := range p.Tasks {
			for _, a := range t.Attachments {
				keep[w.cachePath(p.ID, t.ID, a)] = true
			}
		}
	}
	_ = filepath.WalkDir(w.root(), func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(path, ".json") && !keep[path] {
			_ = os.Remove(path)
		}
		return nil
	})
}
